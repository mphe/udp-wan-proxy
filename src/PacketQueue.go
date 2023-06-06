package main

type PacketQueue struct {
    timeQueue *PriorityQueue[struct{}]
    packetQueue chan []byte
}

func NewPacketQueue(buffer_size int) *PacketQueue {
    return &PacketQueue{
        timeQueue: NewPriorityQueue[struct{}](),
        packetQueue: make(chan []byte, buffer_size),
    }
}

func (pq *PacketQueue) close() {
    close(pq.packetQueue)
}
