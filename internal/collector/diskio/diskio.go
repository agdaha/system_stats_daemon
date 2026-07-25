package diskio

import (
	"context"
	"sort"
	"time"

	"system_stats_deamon/internal/ringbuffer"
)

type Sample struct {
	Device string
	TPS    float32
	KBps   float32
}

type Collector struct {
	buf *ringbuffer.RingBuffer[[]Sample]
}

func New() *Collector {
	return &Collector{buf: ringbuffer.New[[]Sample](300)}
}

func (c *Collector) Start(ctx context.Context) {
	go c.run(ctx)
}

func (c *Collector) Average(window time.Duration) []Sample {
	entries := c.buf.Since(time.Now().Add(-window))
	if len(entries) == 0 {
		return nil
	}

	type accum struct {
		tps  float64
		kbps float64
		n    int
	}
	sums := make(map[string]*accum)

	for _, e := range entries {
		for _, s := range e.Data {
			a := sums[s.Device]
			if a == nil {
				a = &accum{}
				sums[s.Device] = a
			}
			a.tps += float64(s.TPS)
			a.kbps += float64(s.KBps)
			a.n++
		}
	}

	result := make([]Sample, 0, len(sums))
	for dev, a := range sums {
		n := float64(a.n)
		result = append(result, Sample{
			Device: dev,
			TPS:    float32(a.tps / n),
			KBps:   float32(a.kbps / n),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Device < result[j].Device
	})

	return result
}
