package main

import (
    "fmt"
    "time"
    "math/rand"
)

type WAN struct {
    delay time.Duration
    jitter time.Duration
    probPacketLossStart float32
    probPacketLossStop float32
    _isPacketLoss bool
}

func (wan *WAN) NextTimestamp() (timestamp time.Time, drop bool) {
    // Switch between packet-loss state based on given probabilities
    if wan._isPacketLoss {
	if rand.Float32() < wan.probPacketLossStop {
	    wan._isPacketLoss = false
	}
    } else {
	if rand.Float32() < wan.probPacketLossStart {
	    wan._isPacketLoss = true
	}
    }

    if wan._isPacketLoss {
	return time.Now(), true
    }

    ts := time.Now().
	Add(wan.delay).
	Add(time.Duration(float64(wan.jitter) * rand.Float64()))

    return ts, false
}

func (wan *WAN) String() string {
    return fmt.Sprintf(`WAN:
    Delay: %v
    Jitter: %v
    Packet loss:
	Start: %v%%
	Stop: %v%%
    `, wan.delay, wan.jitter, wan.probPacketLossStart * 100.0, wan.probPacketLossStop * 100.0)
}
