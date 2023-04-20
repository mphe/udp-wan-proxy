package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/akamensky/argparse"
)

const RUNNING_LATE_WARN_THRESHOLD = time.Duration(500) * time.Microsecond
const SOCKET_RW_BUFFER_SIZE = 20 * 1024 * 1024  // 20 MB
const SPINLOCK_SLEEP_TIME = time.Duration(1) * time.Microsecond


type PacketQueue = PriorityQueue[[]byte]


func run_listener(wg *sync.WaitGroup, listen_port int, queue *PacketQueue, wan *WAN) {
    defer wg.Done()

    fmt.Println("Starting listener on", listen_port)
    listener, err := net.ListenUDP("udp", &net.UDPAddr { Port: listen_port })

    if err != nil {
        log.Fatal(err)
    }

    listener.SetReadBuffer(SOCKET_RW_BUFFER_SIZE)
    defer listener.Close()

    buf := make([]byte, 4096)

    for {
        n, _, err := listener.ReadFrom(buf)

        if n > 0 {
            targetTime, drop := wan.compute_next_timestamp()
            if drop {
                fmt.Println("X  Packet lost", n)
                continue
            }

            fmt.Println("-> Received:", n)
            data := make([]byte, n)
            copy(data, buf[:n])
            queue.Push(targetTime, data)
        }

        if err != nil {
            log.Fatal(err)
        }
    }
}


func run_sender(wg *sync.WaitGroup, relay_port int, queue *PacketQueue) {
    defer wg.Done()

    fmt.Println("Starting relay to", relay_port)
    sender, err := net.DialUDP("udp", nil, &net.UDPAddr { Port: relay_port })

    if err != nil {
        log.Fatal(err)
    }

    sender.SetWriteBuffer(SOCKET_RW_BUFFER_SIZE)
    defer sender.Close()

    var clock_inaccuracy time.Duration
    var num_sent time.Duration
    var last_time time.Time  // Used to check for packet reordering

    for {
        targetTime := queue.Peek().priority

        if targetTime != last_time {
            fmt.Println("<=> Packet reordered")
        }

        timeDelta := targetTime.Sub(time.Now())  // Ensure time.Now() gets evaluated after Peek()

        if timeDelta < 0 {
            fmt.Println("Target time behind schedule", timeDelta)
        } else {
            select {
            case <-time.After(timeDelta):
            case <-queue.WaitForItemAdded():
                last_time = targetTime
                continue // New packet added, maybe it is scheduled earlier than the current one
            }
        }

        // Theoretically, time.After and ItemAdded could occur simultaneously.
        // In that case, the new packet might be scheduled earlier than the one we're currently
        // waiting for. Hence, we fetch the the top packet again.
        // Correctness:
        // If the new packet is scheduled at the same time or earlier, we don't have to wait
        // again. If it is scheduled later, it does not matter, as we're not dealing with it.
        packet := queue.Pop()

        diff := time.Now().Sub(packet.priority)
        clock_inaccuracy += diff
        num_sent += 1

        if diff > RUNNING_LATE_WARN_THRESHOLD {
            fmt.Println("Running late:", diff)
            fmt.Println("Average clock inaccuracy:", clock_inaccuracy / num_sent)
        }

        _, err := sender.Write(packet.value)
        fmt.Println("<- Sent:", len(packet.value))

        // Write() will cause a "connection refused" error when there is no listener on the
        // other side. We can ignore it.
        if err != nil && !errors.Is(err, syscall.ECONNREFUSED) {
            log.Fatal(err)
        }
    }
}


func main() {
    parser := argparse.NewParser("UDP-Proxy", "UDP WAN proxy")
    listen_port := parser.Int("l", "listen", &argparse.Options{Required: true, Help: "Port to listen on"})
    relay_port := parser.Int("r", "relay", &argparse.Options{Required: true, Help: "Port to relay packets to"})
    delay_seconds := parser.Float("d", "delay", &argparse.Options{Required: false, Help: "Packet delay in seconds", Default: 0.0})
    jitter_seconds := parser.Float("j", "jitter", &argparse.Options{Required: false, Help: "Random packet jitter in seconds", Default: 0.0})
    probPacketLossStart := parser.Float("", "loss-start", &argparse.Options{Required: false, Help: "Probability for a packet loss phase to occur", Default: 0.0})
    probPacketLossStop := parser.Float("", "loss-stop", &argparse.Options{Required: false, Help: "Probability for a packet loss phase to end", Default: 0.0})

    err := parser.Parse(os.Args)

    if err != nil {
        fmt.Println(parser.Usage(nil))
        log.Fatal(err)
    }

    fmt.Println("Version", runtime.Version())
    fmt.Println("NumCPU", runtime.NumCPU())
    fmt.Println("GOMAXPROCS", runtime.GOMAXPROCS(0))
    fmt.Println()

    var pq *PacketQueue = NewPriorityQueue[[]byte]()
    var wg sync.WaitGroup
    wan := NewWAN(*jitter_seconds, *delay_seconds)
    wan.probPacketLossStart = float32(*probPacketLossStart)
    wan.probPacketLossStop = float32(*probPacketLossStop)

    fmt.Println(wan)

    wg.Add(1)
    go run_listener(&wg, *listen_port, pq, wan)
    wg.Add(1)
    go run_sender(&wg, *relay_port, pq)

    wg.Wait()
}
