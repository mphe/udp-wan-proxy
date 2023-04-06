package main

import (
    "fmt"
    "time"
)

type WAN struct {
    delay time.Duration
    jitter time.Duration
}

func (wan *WAN) compute_send_timestamp() time.Time {
    return time.Now().Add(wan.delay)
}

func (wan *WAN) String() string {
    return fmt.Sprintf(`WAN:
    Delay: %v
    Jitter: %v`, wan.delay, wan.jitter)
}
