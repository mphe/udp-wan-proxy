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


type PacketEntry struct {
    data []byte
    targetTime time.Time
}

type PacketQueue = chan PacketEntry


func run_listener(wg *sync.WaitGroup, listen_port int, queue PacketQueue, wan *WAN) {
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
            fmt.Println("-> Received:", n)

            targetTime, drop := wan.compute_next_timestamp()
            if drop {
                fmt.Println("X Packet lost")
                continue
            }

            data := make([]byte, n)
            copy(data, buf[:n])

            queue <- PacketEntry{
                data: data,
                targetTime: targetTime,
            }
        }

        if err != nil {
            log.Fatal(err)
        }
    }
}


// Sleep for "duration" in intervals of "resolution".
func spinlockSleep(duration time.Duration, resolution time.Duration) {
    targetTime := time.Now().Add(duration)

    for duration > 0 {
        sleep := resolution
        if duration < sleep {
            sleep = duration
        }
        time.Sleep(sleep)
        duration = targetTime.Sub(time.Now())
    }
}


func run_sender(wg *sync.WaitGroup, relay_port int, queue PacketQueue) {
    defer wg.Done()

    fmt.Println("Starting relay to", relay_port)
    sender, err := net.DialUDP("udp", nil, &net.UDPAddr { Port: relay_port })

    if err != nil {
        log.Fatal(err)
    }

    sender.SetWriteBuffer(SOCKET_RW_BUFFER_SIZE)
    defer sender.Close()

    num_sent := time.Duration(0)
    clock_inaccuracy := time.Duration(0)

    for {
        packet := <- queue
        timeDelta := packet.targetTime.Sub(time.Now())

        // | Method                | Avg lateness | Notes                                         |
        // |-----------------------+--------------+-----------------------------------------------|
        // | Single sleep          | ~500µs       | Avg CPU usage ~4%                             |
        // | Spinlock, 1µs sleep   | ~200µs       | Avg CPU usage ~8%                             |
        // | Spinlock, 100ns sleep | ~15µs        | Avg CPU usage ~19%                            |
        // | Spinlock, while true  | ~90ns - 1µs  | Avg CPU usage ~14%, but one CPU core on 100%. |

        if timeDelta < 0 {
            fmt.Println("Target time behind schedule", timeDelta)
        } else {
            // Sleep spinlock
            spinlockSleep(timeDelta, SPINLOCK_SLEEP_TIME)

            // Single sleep
            // time.Sleep(timeDelta)

            // Spinlock
            // for timeDelta > 0 {
            //     timeDelta = packet.targetTime.Sub(time.Now())
            // }

            diff := time.Now().Sub(packet.targetTime)
            clock_inaccuracy += diff
            num_sent += 1

            if diff > RUNNING_LATE_WARN_THRESHOLD {
                fmt.Println("Running late:", diff)
                fmt.Println("Average clock inaccuracy:", clock_inaccuracy / num_sent)
            }

        }
        _, err := sender.Write(packet.data)
        fmt.Println("<- Sent:", len(packet.data))

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

    queue := make(PacketQueue)
    var wg sync.WaitGroup
    wan := NewWAN(*jitter_seconds, *delay_seconds)
    wan.probPacketLossStart = float32(*probPacketLossStart)
    wan.probPacketLossStop = float32(*probPacketLossStop)

    fmt.Println(wan)

    wg.Add(1)
    go run_listener(&wg, *listen_port, queue, wan)
    wg.Add(1)
    go run_sender(&wg, *relay_port, queue)

    wg.Wait()
}
