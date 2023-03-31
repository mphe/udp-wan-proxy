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


type QueueData struct {
    target_time float32
    data []byte
}


type PacketQueue = PriorityQueue[[]byte]


func run_listener(wg *sync.WaitGroup, listen_addr *string, queue *PacketQueue, wan *WAN) {
    defer wg.Done()

    fmt.Println("Starting listener on", *listen_addr)
    listener, err := net.ListenPacket("udp", *listen_addr)

    if err != nil {
        log.Fatal(err)
    }

    defer listener.Close()

    for {
        buf := make([]byte, 4096)
        n, _, err := listener.ReadFrom(buf)

        if n > 0 {
            data := make([]byte, n)
            copy(data, buf[:n])
            fmt.Println("Received:", len(data))
            queue.Push(wan.compute_send_timestamp(), data)
        }

        if err != nil {
            log.Fatal(err)
        }
    }
}


func run_sender(wg *sync.WaitGroup, relay_addr *string, queue *PacketQueue) {
    defer wg.Done()

    fmt.Println("Starting relay to", *relay_addr)
    sender, err := net.Dial("udp", *relay_addr)

    if err != nil {
        log.Fatal(err)
    }

    defer sender.Close()

    for {
        if queue.IsEmpty() {
            fmt.Println("Waiting for queue data")
            <-queue.ItemAdded
            fmt.Println("Finished waiting")
        }

        fmt.Println("Peek")
        timeDelta := queue.Peek().priority.Sub(time.Now())
        fmt.Println("Peek done")

        if timeDelta > 0 {
            fmt.Println("Waiting ", timeDelta)

            select {
            case <-time.After(timeDelta):
            case <- queue.ItemAdded:
                fmt.Println("Item Added -> restart")
                continue // New packet added, maybe it is scheduled earlier than the current one
            }
        }

        // Theoretically, time.After and ItemAdded could occur simultaneously.
        // In that case, the new packet might be scheduled earlier than the one we're currently
        // waiting for. Hence, we fetch the the top packet again.
        // Correctness:
        // If the new packet is scheduled at the same time or earlier, we don't have to wait
        // again, obviously. If it is scheduled later, we're fine.
        fmt.Println("Pop")
        packet := queue.Pop()
        fmt.Println("Pop done")

        diff := time.Now().Sub(packet.priority)
        if diff > RUNNING_LATE_WARN_THRESHOLD {
            log.Println("Running late:", diff.Microseconds(), "µs")
        }

        _, err := sender.Write(packet.value)
        fmt.Println("Sent:", len(packet.value))

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

    err := parser.Parse(os.Args)

    if err != nil {
        log.Println(parser.Usage(nil))
        log.Fatal(err)
    }

    fmt.Println("Version", runtime.Version())
    fmt.Println("NumCPU", runtime.NumCPU())
    fmt.Println("GOMAXPROCS", runtime.GOMAXPROCS(0))

    listen_addr := fmt.Sprintf(":%v", *listen_port)
    relay_addr := fmt.Sprintf(":%v", *relay_port)

    fmt.Println("Relay address:", relay_addr)

    var pq *PacketQueue = NewPriorityQueue[[]byte](-1)
    var wg sync.WaitGroup
    wan := WAN{
        delay: time.Duration(*delay_seconds * float64(time.Second)),
        jitter: time.Duration(*jitter_seconds * float64(time.Second)),
    }

    fmt.Println(&wan)

    wg.Add(1)
    go run_listener(&wg, &listen_addr, pq, &wan)
    wg.Add(1)
    go run_sender(&wg, &relay_addr, pq)

    wg.Wait()
}
