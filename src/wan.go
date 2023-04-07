package main

import (
    "fmt"
    "time"
    "math/rand"
)

type WAN struct {
    delay time.Duration
    jitter time.Duration
    jitterNs float64
}

func NewWAN(jitterSeconds float64, delaySeconds float64) *WAN {
    nsPerSecond := float64(time.Second)
    return &WAN{
        delay: time.Duration(delaySeconds * nsPerSecond),
        jitter: time.Duration(jitterSeconds * nsPerSecond),
	jitterNs: jitterSeconds * nsPerSecond,
    }
}

func (wan *WAN) compute_send_timestamp() time.Time {
    return time.Now().
	Add(wan.delay).
	Add(time.Duration(wan.jitterNs * rand.Float64()))
}

func (wan *WAN) String() string {
    return fmt.Sprintf(`WAN:
    Delay: %v
    Jitter: %v`, wan.delay, wan.jitter)
}
