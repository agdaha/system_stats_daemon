package loadavg

import (
	"context"
	"time"

	"system_stats_deamon/internal/ringbuffer"
)

type Sample struct {
	One     float32
	Five    float32
	Fifteen float32
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
		sum.One += e.Data.One
		sum.Five += e.Data.Five
		sum.Fifteen += e.Data.Fifteen
	}

	n := float32(len(entries))
	return Sample{
		One:     sum.One / n,
		Five:    sum.Five / n,
		Fifteen: sum.Fifteen / n,
	}, true
}
