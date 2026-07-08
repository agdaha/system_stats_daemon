package cpu

import (
	"context"
	"time"

	"system_stats_deamon/internal/ringbuffer"
)

type Sample struct {
	User   float32
	System float32
	Idle   float32
}

type Collector struct {
	buf *ringbuffer.RingBuffer[Sample]
}

func New() *Collector {
	return &Collector{buf: ringbuffer.New[Sample](300)}
}

func (c *Collector) Start(ctx context.Context) {
	go c.run(ctx)
}

func (c *Collector) Average(window time.Duration) (Sample, bool) {
	entries := c.buf.Since(time.Now().Add(-window))
	if len(entries) == 0 {
		return Sample{}, false
	}

	var sum Sample
	for _, e := range entries {
		sum.User += e.Data.User
		sum.System += e.Data.System
		sum.Idle += e.Data.Idle
	}

	n := float32(len(entries))
	return Sample{
		User:   sum.User / n,
		System: sum.System / n,
		Idle:   sum.Idle / n,
	}, true
}
