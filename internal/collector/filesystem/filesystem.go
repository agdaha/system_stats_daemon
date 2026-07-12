package filesystem

import (
	"context"
	"time"

	"system_stats_deamon/internal/ringbuffer"
)

type Sample struct {
	Filesystem        string
	MountPoint        string
	UsedMB            uint64
	UsedPercent       float32
	UsedInodes        uint64
	UsedInodesPercent float32
}

type Collector struct {
	buf *ringbuffer.RingBuffer[[]Sample]
}

func New() *Collector {
	return &Collector{buf: ringbuffer.New[[]Sample](120)}
}

func (c *Collector) Start(ctx context.Context) {
	go c.run(ctx)
}

func (c *Collector) Average(window time.Duration) []Sample {
	entries := c.buf.Since(time.Now().Add(-window))
	if len(entries) == 0 {
		return nil
	}
	return entries[len(entries)-1].Data
}
