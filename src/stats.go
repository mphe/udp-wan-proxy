package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Statistics struct {
    sent int32
    received int32
    lost int32
    sentBytes int32
    receivedBytes int32
    clockDeltaNs int64
}


func (stats *Statistics) Sent(numBytes int, clockDelta time.Duration) {
    atomic.AddInt32(&stats.sent, 1)
    atomic.AddInt32(&stats.sentBytes, int32(numBytes))
    atomic.AddInt64(&stats.clockDeltaNs, clockDelta.Nanoseconds())
}


func (stats *Statistics) Received(numBytes int) {
    atomic.AddInt32(&stats.received, 1)
    atomic.AddInt32(&stats.receivedBytes, int32(numBytes))
}


func (stats *Statistics) Lost() {
    atomic.AddInt32(&stats.lost, 1)
}


func (stats *Statistics) Reset() {
    atomic.StoreInt32(&stats.sent, 0)
    atomic.StoreInt32(&stats.received, 0)
    atomic.StoreInt32(&stats.lost, 0)
    atomic.StoreInt32(&stats.sentBytes, 0)
    atomic.StoreInt32(&stats.receivedBytes, 0)
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
            fmt.Printf("Avg. clock inaccuracy: %v\n", time.Duration(stats.clockDeltaNs / int64(stats.sent)))
            fmt.Println("--------------------------------------------")
            stats.Reset()
        }
    }()
}
