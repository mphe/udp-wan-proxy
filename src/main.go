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


type PacketEntry struct {
    data []byte
    targetTime time.Time
}

type PacketQueue = chan PacketEntry


func run_listener(wg *sync.WaitGroup, listen_addr *string, queue PacketQueue, wan *WAN) {
    defer wg.Done()

    fmt.Println("Starting listener on", *listen_addr)
    listener, err := net.ListenPacket("udp", *listen_addr)

    if err != nil {
        log.Fatal(err)
    }

    defer listener.Close()

    for {
        buf := make([]byte, 2048)
        n, _, err := listener.ReadFrom(buf)

        if n > 0 {
            fmt.Println("Received:", n)
            data := make([]byte, n)
            copy(data, buf[:n])
            queue <- PacketEntry{
                data: data,
                targetTime: wan.compute_send_timestamp(),
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


func run_sender(wg *sync.WaitGroup, relay_addr *string, queue PacketQueue) {
    defer wg.Done()

    fmt.Println("Starting relay to", *relay_addr)
    sender, err := net.Dial("udp", *relay_addr)

    if err != nil {
        log.Fatal(err)
    }

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
            spinlockSleep(timeDelta, time.Duration(1) * time.Microsecond)

            // Single sleep
            // time.Sleep(timeDelta)

            // Spinlock
            // for timeDelta > 0 {
            //     timeDelta = packet.targetTime.Sub(time.Now())
            // }

            diff := time.Now().Sub(packet.targetTime)
            clock_inaccuracy += diff
            num_sent += 1
            fmt.Println("Average clock inaccuracy:", clock_inaccuracy / num_sent)

            if diff > RUNNING_LATE_WARN_THRESHOLD {
                fmt.Println("Running late:", diff)
            }

        }
        _, err := sender.Write(packet.data)
        fmt.Println("Sent:", len(packet.data))

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
        fmt.Println(parser.Usage(nil))
        log.Fatal(err)
    }

    fmt.Println("Version", runtime.Version())
    fmt.Println("NumCPU", runtime.NumCPU())
    fmt.Println("GOMAXPROCS", runtime.GOMAXPROCS(0))

    listen_addr := fmt.Sprintf(":%v", *listen_port)
    relay_addr := fmt.Sprintf(":%v", *relay_port)

    fmt.Println("Relay address:", relay_addr)

    queue := make(PacketQueue)
    var wg sync.WaitGroup
    wan := NewWAN(*jitter_seconds, *delay_seconds)

    fmt.Println(wan)

    wg.Add(1)
    go run_listener(&wg, &listen_addr, queue, wan)
    wg.Add(1)
    go run_sender(&wg, &relay_addr, queue)

    wg.Wait()
}
