package main

import (
    "fmt"
    "time"
    "math/rand"
)

type WAN struct {
    delay time.Duration
    jitter time.Duration
    _jitterNs float64
    probPacketLossStart float32
    probPacketLossStop float32
    _isPacketLoss bool
}

func NewWAN(jitterSeconds float64, delaySeconds float64) *WAN {
    nsPerSecond := float64(time.Second)
    return &WAN{
        delay: time.Duration(delaySeconds * nsPerSecond),
        jitter: time.Duration(jitterSeconds * nsPerSecond),
	_jitterNs: jitterSeconds * nsPerSecond,
    }
}

func (wan *WAN) compute_next_timestamp() (timestamp time.Time, drop bool) {
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
	Add(time.Duration(wan._jitterNs * rand.Float64()))

    return ts, false
}

func (wan *WAN) String() string {
    return fmt.Sprintf(`WAN:
    Delay: %v
    Jitter: %v
    Packet loss:
	Start: %v
	Stop: %v
    `, wan.delay, wan.jitter, wan.probPacketLossStart, wan.probPacketLossStop)
}
