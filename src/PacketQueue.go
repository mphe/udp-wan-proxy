package main

type PacketQueue struct {
    timeQueue *PriorityQueue[struct{}]
    packetQueue chan []byte
}

func NewPacketQueue() *PacketQueue {
    return &PacketQueue{
        timeQueue: NewPriorityQueue[struct{}](),
        packetQueue: make(chan []byte),
    }
}
