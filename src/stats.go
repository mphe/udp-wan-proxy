package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Go does not have a Min() function for integers in its stdlib...
func Max(a, b int64) int64 {
    if a > b {
        return a
    }
    return b
}

type Statistics struct {
    sent int64
    received int64
    lost int64
    sentBytes int64
    receivedBytes int64
    clockDeltaNs int64
}


func (stats *Statistics) Sent(numBytes int, clockDelta time.Duration) {
    atomic.AddInt64(&stats.sent, 1)
    atomic.AddInt64(&stats.sentBytes, int64(numBytes))
    atomic.AddInt64(&stats.clockDeltaNs, clockDelta.Nanoseconds())
}


func (stats *Statistics) Received(numBytes int) {
    atomic.AddInt64(&stats.received, 1)
    atomic.AddInt64(&stats.receivedBytes, int64(numBytes))
}


func (stats *Statistics) Lost() {
    atomic.AddInt64(&stats.lost, 1)
}


func (stats *Statistics) Reset() {
    atomic.StoreInt64(&stats.sent, 0)
    atomic.StoreInt64(&stats.received, 0)
    atomic.StoreInt64(&stats.lost, 0)
    atomic.StoreInt64(&stats.sentBytes, 0)
    atomic.StoreInt64(&stats.receivedBytes, 0)
    atomic.StoreInt64(&stats.clockDeltaNs, 0)
}


func (stats *Statistics) StartWatchThread(wg *sync.WaitGroup, interval time.Duration) {
    go func() {
        defer wg.Done()

        for {
            time.Sleep(interval)

            if stats.sent == 0 && stats.received == 0 && stats.lost == 0 {
                continue
            }

            fmt.Println("---------------- Statistics ----------------")
            fmt.Println("Interval: ", interval)
            fmt.Printf("Sent:      %v packets, %v KiB\n", stats.sent, float32(stats.sentBytes) / 1024)
            fmt.Printf("Received:  %v packets, %v KiB\n", stats.received, float32(stats.receivedBytes) / 1024)
            fmt.Printf("Lost:      %v packets\n", stats.lost)
            fmt.Printf("Avg. clock inaccuracy: %v\n", time.Duration(stats.clockDeltaNs / Max(1, stats.sent)))
            fmt.Println("--------------------------------------------")
            stats.Reset()
        }
    }()
}
