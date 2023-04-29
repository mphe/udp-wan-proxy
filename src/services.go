package main

import (
    "errors"
    "fmt"
    "log"
    "net"
    "sync"
    "syscall"
    "time"
)

const RUNNING_LATE_WARN_THRESHOLD = time.Duration(500) * time.Microsecond
const SOCKET_RW_BUFFER_SIZE = 20 * 1024 * 1024  // 20 MB
const SPINLOCK_SLEEP_TIME = time.Duration(100) * time.Nanosecond


func run_listener(wg *sync.WaitGroup, listen_port int, queue *PacketQueue, wan *WAN, stats *Statistics) {
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
                stats.Lost()
                continue
            }

            stats.Received(n)
            data := make([]byte, n)
            copy(data, buf[:n])
            queue.timeQueue.Push(targetTime, struct{}{})
            queue.packetQueue <- data
        }

        if err != nil {
            log.Fatal(err)
        }
    }
}


func run_sender(wg *sync.WaitGroup, relay_port int, queue *PacketQueue, stats *Statistics) {
    defer wg.Done()

    fmt.Println("Starting relay to", relay_port)
    sender, err := net.DialUDP("udp", nil, &net.UDPAddr { Port: relay_port })

    if err != nil {
        log.Fatal(err)
    }

    sender.SetWriteBuffer(SOCKET_RW_BUFFER_SIZE)
    defer sender.Close()

    for {
        targetTime := queue.timeQueue.Peek().priority
        timeDelta := time.Until(targetTime)  // Ensure time.Now() gets evaluated after Peek()

        if timeDelta < 0 {
            fmt.Println("Target time behind schedule", timeDelta)
        } else {
            if !spinSleep(timeDelta, SPINLOCK_SLEEP_TIME, queue.timeQueue.WaitForItemAdded()) {
                continue  // New timestamp added, maybe it is scheduled earlier than the current one
            }
        }

        // Finished waiting, pop the timestamp.
        // Theoretically, time.After and ItemAdded could occur simultaneously.
        // In that case, the new packet might be scheduled earlier than the one we're currently
        // waiting for. Hence, we fetch the the top packet again.
        // Correctness:
        // If the new packet is scheduled at the same time or earlier, we don't have to wait
        // again. If it is scheduled later, it does not matter, as we're not dealing with it.
        timestamp := queue.timeQueue.Pop().priority

        // Compute clock inaccuracy
        clockDiff := time.Since(timestamp)

        if clockDiff > RUNNING_LATE_WARN_THRESHOLD {
            fmt.Println("Running late:", clockDiff, ", waited for", timeDelta)
        }

        data := <- queue.packetQueue
        _, err := sender.Write(data)
        stats.Sent(len(data), clockDiff)

        // Write() will cause a "connection refused" error when there is no listener on the
        // other side. We can ignore it.
        if err != nil && !errors.Is(err, syscall.ECONNREFUSED) {
            log.Fatal(err)
        }
    }
}


// Sleep for "duration" in intervals of "sleepInterval". interrupt_chan is a channel that can be
// written to break the sleep and return.
// Returns false when it was interrupted, true otherwise.
func spinSleep(duration time.Duration, sleepInterval time.Duration, interrupt_chan <-chan struct{}) bool {
    targetTime := time.Now().Add(duration)

    for duration > 0 {
        sleep := sleepInterval
        if duration < sleep {
            sleep = duration
        }

        if interrupt_chan == nil {
            time.Sleep(sleep)
        } else {
            select {
            case <-time.After(sleep):
            case <-interrupt_chan:
                return false
            }
        }

        duration = time.Until(targetTime)
    }

    return true
}
