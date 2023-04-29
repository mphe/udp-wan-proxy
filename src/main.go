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

    wan := NewWAN(*jitter_seconds, *delay_seconds)
    wan.probPacketLossStart = float32(*probPacketLossStart)
    wan.probPacketLossStop = float32(*probPacketLossStop)
    fmt.Println(wan)

    pq := NewPriorityQueue[[]byte]()
    stats := Statistics{}

    var wg sync.WaitGroup
    wg.Add(1)
    stats.StartWatchThread(&wg, time.Duration(1) * time.Second)
    wg.Add(1)
    go run_listener(&wg, *listen_port, pq, wan, &stats)
    wg.Add(1)
    go run_sender(&wg, *relay_port, pq, &stats)

    wg.Wait()
}
