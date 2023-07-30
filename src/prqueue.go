package main

import (
	"container/heap"
	"fmt"
	"sync"
	"time"
)

type PriorityT = time.Time

type Item[T any] struct {
    priority PriorityT
    value    T
}

func (item Item[T]) String() string {
    return fmt.Sprintf("priority: %v,  value: %v", item.priority, item.value)
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

////////////////////////////////////////////////////////////

type PriorityQueue[T any] struct {
    mu sync.RWMutex
    items Heap[T]
    itemAdded chan struct{}
    headPriorityChanged chan struct{}
}

// Returns and initializes a new priority queue. Smaller values have higher priority.
func NewPriorityQueue[T any]() *PriorityQueue[T] {
    pq := &PriorityQueue[T]{
        itemAdded: make(chan struct{}),
        headPriorityChanged: make(chan struct{}),
    }
    return pq
}

// Pushes an item onto the priority queue.
func (pq *PriorityQueue[T]) Push(priority PriorityT, data T) {
    pq.mu.Lock()
    defer pq.mu.Unlock()

    var headPrio time.Time

    if (pq.items.Len() > 0) {
        headPrio = pq.items[0].priority
    }

    heap.Push(&pq.items, Item[T]{priority, data})

    // Check if priority of first element changed
    if pq.items.Len() == 1 || pq.items[0].priority != headPrio {
        pq._TriggerBroadcast(&pq.headPriorityChanged)
    }

    pq._TriggerBroadcast(&pq.itemAdded)
}

// Pops the top item from the priority queue. Blocks if the queue is empty.
func (pq *PriorityQueue[T]) Pop() Item[T] {
    pq.mu.Lock()
    defer pq.mu.Unlock()
    pq._WaitForData(&pq.mu)
    ret := heap.Pop(&pq.items).(Item[T])
    pq._TriggerBroadcast(&pq.headPriorityChanged)
    return ret
}

// Returns the top item from the priority queue but does not remove it. Blocks if the queue is empty.
func (pq *PriorityQueue[T]) Peek() Item[T] {
    pq.mu.RLock()
    defer pq.mu.RUnlock()
    pq._WaitForData(pq.mu.RLocker())
    // Return a copy to maintain thread safety that wouldn't be guaranteed with a pointer.
    return pq.items[0]
}

func (pq *PriorityQueue[T]) Len() int {
    pq.mu.RLock()
    defer pq.mu.RUnlock()
    return pq.items.Len()
}

func (pq *PriorityQueue[T]) IsEmpty() bool {
    return pq.Len() == 0
}


// Returns a channel that can be listened on to wait until the next item is added.
func (pq *PriorityQueue[T]) WaitForItemAdded() <-chan struct{} {
    pq.mu.RLock()
    defer pq.mu.RUnlock()
    return pq.itemAdded
}


// Returns a channel that can be listened on to wait until the priority of the topmost item to change.
func (pq *PriorityQueue[T]) WaitForHeadPriorityChange() <-chan struct{} {
    pq.mu.RLock()
    defer pq.mu.RUnlock()
    return pq.headPriorityChanged
}


func (pq *PriorityQueue[T]) _TriggerBroadcast(c *chan struct{}) {
    // First make a new channel to prevent other code from accessing a closed channel through
    // WaitFor*() functions.
    old := *c
    *c = make(chan struct{})

    // Afterwards broadcast the signal by closing the channel.
    close(old)
}


// Works like sync.Cond.Wait(). Acquire pq.mu, then call _WaitForData().
func (pq *PriorityQueue[T]) _WaitForData(locker sync.Locker) {
    // We can't use pq.IsEmpty() here, because it would require a read lock in Len() but we can't
    // require a read lock if holding a write lock.
    for pq.items.Len() == 0 {
        locker.Unlock()
        <-pq.WaitForItemAdded()
        locker.Lock()
    }
}
