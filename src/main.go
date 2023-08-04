package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
	"github.com/akamensky/argparse"
)

const MAX_PACKET_QUEUE_SIZE int = 4096  // 4096 packets should suffice in pretty much all cases

func main() {
    parser := argparse.NewParser("UDP-Proxy", "UDP WAN proxy")
    listen_port := parser.Int("l", "listen", &argparse.Options{Required: true, Help: "Port to listen on"})
    relay_port := parser.Int("r", "relay", &argparse.Options{Required: true, Help: "Port to relay packets to"})
    delay_ms := parser.Int("d", "delay", &argparse.Options{Required: false, Help: "Packet delay in milliseconds", Default: 0})
    jitter_ms := parser.Int("j", "jitter", &argparse.Options{Required: false, Help: "Random packet jitter in milliseconds", Default: 0})
    probPacketLossStart := parser.Float("", "loss-start", &argparse.Options{Required: false, Help: "Probability for a packet loss phase to occur (0.0 - 1.0)", Default: 0.0})
    probPacketLossStop := parser.Float("", "loss-stop", &argparse.Options{Required: false, Help: "Probability for a packet loss phase to end (0.0 - 1.0)", Default: 0.0})
    csvFile := parser.String("", "csv", &argparse.Options{Required: false, Help: "Output CSV file for stats"})
    spinSleepNs := parser.Int("s", "sleep-interval", &argparse.Options{Required: false, Help: "Sleep interval in nanoseconds", Default: int(DEFAULT_SPINLOCK_SLEEP_TIME)})

    err := parser.Parse(os.Args)

    if err != nil {
        fmt.Println(parser.Usage(nil))
        log.Fatal(err)
    }

    runtime.GOMAXPROCS(runtime.NumCPU())
    fmt.Println("Version", runtime.Version())
    fmt.Println("NumCPU", runtime.NumCPU())
    fmt.Println("GOMAXPROCS", runtime.GOMAXPROCS(0))
    fmt.Println()

    wan := WAN {
	delay: time.Duration(*delay_ms) * time.Millisecond,
	jitter: time.Duration(*jitter_ms) * time.Millisecond,
	probPacketLossStart: float32(*probPacketLossStart),
	probPacketLossStop: float32(*probPacketLossStop),
    }
    fmt.Println(&wan)

    pq := NewPacketQueue(MAX_PACKET_QUEUE_SIZE)
    stats := Statistics{}

    if csvFile != nil && len(*csvFile) > 0{
	stats.LogToCSV(*csvFile)
    }

    var wg sync.WaitGroup
    wg.Add(1)
    stats.StartWatchThread(&wg, time.Duration(1) * time.Second)
    wg.Add(1)
    go RunListener(&wg, *listen_port, pq, &wan, &stats)
    wg.Add(1)
    go RunSender(&wg, *relay_port, pq, &stats, time.Duration(*spinSleepNs))

    wg.Wait()
    pq.Close()
}
