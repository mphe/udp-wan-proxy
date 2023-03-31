package main

import (
	"container/heap"
	"sync"
	"time"
)

type PriorityT = time.Time

type Item[T any] struct {
    priority PriorityT
    value    T
}


type Heap[T any] []Item[T]

func (pq *Heap[T]) Len() int {
    return len(*pq)
}

func (pq *Heap[T]) Less(i, j int) bool {
    return (*pq)[i].priority.Before((*pq)[j].priority)
}

func (pq *Heap[T]) Swap(i, j int) {
    (*pq)[i], (*pq)[j] = (*pq)[j], (*pq)[i]
}

func (pq *Heap[T]) Push(item any) {
    *pq = append(*pq, item.(Item[T]))
}

func (pq *Heap[T]) Pop() any {
    n := len(*pq)
    item := (*pq)[n - 1]
    *pq = (*pq)[:n - 1]
    return item
}


type PriorityQueue[T any] struct {
    mu sync.RWMutex
    items Heap[T]
    ItemAdded chan int
}

// NewPriorityQueue returns a new priority queue with the item's cap set at capacity; if capacity > 0.
func NewPriorityQueue[T any](capacity int) *PriorityQueue[T] {
    if capacity <= 0 {
        return &PriorityQueue[T]{ItemAdded: make(chan int, 1)}
    }
    return &PriorityQueue[T]{
        items: make([]Item[T], 0, capacity),
        ItemAdded: make(chan int),
    }
}

func (pq *PriorityQueue[T]) Len() int {
    pq.mu.RLock()
    defer pq.mu.RUnlock()
    return pq.items.Len()
}

// Pushes an item onto the priority queue.
func (pq *PriorityQueue[T]) Push(priority PriorityT, data T) {
    pq.mu.Lock()
    // defer pq.mu.Unlock()
    heap.Push(&pq.items, Item[T]{priority, data})
    pq.mu.Unlock()
    pq.ItemAdded <- 0
}

// Pop pops the next item from the priority queue.
func (pq *PriorityQueue[T]) Pop() Item[T] {
    pq.mu.Lock()
    defer pq.mu.Unlock()
    return pq.items.Pop().(Item[T])
}

func (pq *PriorityQueue[T]) Peek() Item[T] {
    pq.mu.RLock()
    defer pq.mu.RUnlock()
    // Return a copy to maintain thread safety that wouldn't be guaranteed with a pointer.
    return pq.items[len(pq.items) - 1]
}

func (pq *PriorityQueue[T]) IsEmpty() bool {
    return pq.Len() == 0
}

// Wait until the queue is non-empty.
func (pq *PriorityQueue[T]) WaitForData() {
    // Ensure nothing gets added while checking
    pq.mu.RLock()

    if !pq.IsEmpty() {
        pq.mu.RUnlock()
        return
    }

    // Remove last flag (if possible)
    select {
        case <-pq.ItemAdded:
        default:
    }

    pq.mu.RUnlock()

    // Wait
    <-pq.ItemAdded
}
