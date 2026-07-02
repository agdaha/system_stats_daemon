package ringbuffer

import (
	"sync"
	"time"
)

type Entry[T any] struct {
	Time time.Time
	Data T
}

type RingBuffer[T any] struct {
	mu    sync.Mutex
	items []Entry[T]
	cap   int
	head  int
	count int
}

func New[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{
		items: make([]Entry[T], capacity),
		cap:   capacity,
	}
}

func (rb *RingBuffer[T]) Push(data T) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.items[rb.head] = Entry[T]{Time: time.Now(), Data: data}
	rb.head = (rb.head + 1) % rb.cap
	if rb.count < rb.cap {
		rb.count++
	}
}

func (rb *RingBuffer[T]) Since(t time.Time) []Entry[T] {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	result := make([]Entry[T], 0, rb.count)
	for i := 0; i < rb.count; i++ {
		idx := (rb.head - rb.count + i + rb.cap) % rb.cap
		e := rb.items[idx]
		if !e.Time.Before(t) {
			result = append(result, e)
		}
	}
	return result
}
