package netsockets

import (
	"context"
	"time"

	"system_stats_deamon/internal/ringbuffer"
)

type ListenSample struct {
	Command  string
	PID      uint32
	Protocol string
	Port     uint32
}

type StateSample struct {
	State string
	Count uint32
}

type socketSnapshot struct {
	sockets []ListenSample
	states  []StateSample
}

type Collector struct {
	buf *ringbuffer.RingBuffer[socketSnapshot]
}

func New() *Collector {
	return &Collector{buf: ringbuffer.New[socketSnapshot](120)}
}

func (c *Collector) Start(ctx context.Context) {
	go c.run(ctx)
}

func (c *Collector) Snapshot(window time.Duration) (sockets []ListenSample, states []StateSample) {
	entries := c.buf.Since(time.Now().Add(-window))
	if len(entries) == 0 {
		return nil, nil
	}
	last := entries[len(entries)-1].Data
	return last.sockets, last.states
}
