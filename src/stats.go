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

        intervalS := float32(interval / time.Second)

        for {
            time.Sleep(interval)

            if stats.sent == 0 && stats.received == 0 && stats.lost == 0 {
                continue
            }

            sentKB := float32(stats.sentBytes) / 1024
            recvKB := float32(stats.receivedBytes) / 1024
            sentKb := (sentKB * 8) / intervalS
            recvKb := (recvKB * 8) / intervalS
            sentMb := sentKb / 1024
            recvMb := recvKb / 1024

            fmt.Println("---------------- Statistics ----------------")
            fmt.Println("Interval: ", interval)
            fmt.Printf("Sent:      %v packets, %v KB, %v Kb/s, %v Mb/s\n", stats.sent, sentKB, sentKb, sentMb)
            fmt.Printf("Received:  %v packets, %v KB, %v Kb/s, %v Mb/s\n", stats.received, recvKB, recvKb, recvMb)
            fmt.Printf("Lost:      %v packets\n", stats.lost)
            fmt.Printf("Avg. clock inaccuracy: %v\n", time.Duration(stats.clockDeltaNs / Max(1, stats.sent)))
            fmt.Println("--------------------------------------------")
            stats.Reset()
        }
    }()
}
