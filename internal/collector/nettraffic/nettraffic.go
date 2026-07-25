package nettraffic

import (
	"context"
	"sort"
	"time"

	"system_stats_deamon/internal/ringbuffer"
)

type ProtocolSample struct {
	Interface   string
	BytesPerSec uint64
	Percent     float32
}

type FlowSample struct {
	SrcAddr  string
	DstAddr  string
	Protocol string
	Bps      float32
}

type Collector struct {
	protoBuf *ringbuffer.RingBuffer[[]ProtocolSample]
	flowBuf  *ringbuffer.RingBuffer[[]FlowSample]
}

func New() *Collector {
	return &Collector{
		protoBuf: ringbuffer.New[[]ProtocolSample](300),
		flowBuf:  ringbuffer.New[[]FlowSample](60),
	}
}

func (c *Collector) Start(ctx context.Context) {
	go c.runProto(ctx)
	go c.runFlows(ctx)
}

func (c *Collector) Snapshot(window time.Duration) (protocols []ProtocolSample, flows []FlowSample) {
	since := time.Now().Add(-window)

	protoEntries := c.protoBuf.Since(since)
	if len(protoEntries) > 0 {
		snapshots := make([][]ProtocolSample, len(protoEntries))
		for i, e := range protoEntries {
			snapshots[i] = e.Data
		}
		protocols = averageProtocols(snapshots)
	}

	flowEntries := c.flowBuf.Since(since)
	if len(flowEntries) > 0 {
		flows = flowEntries[len(flowEntries)-1].Data
	}

	return protocols, flows
}

func averageProtocols(snapshots [][]ProtocolSample) []ProtocolSample {
	type accum struct {
		bytes uint64
		n     int
	}
	sums := make(map[string]*accum)

	for _, snap := range snapshots {
		for _, s := range snap {
			a := sums[s.Interface]
			if a == nil {
				a = &accum{}
				sums[s.Interface] = a
			}
			a.bytes += s.BytesPerSec
			a.n++
		}
	}

	result := make([]ProtocolSample, 0, len(sums))
	var totalBytes uint64
	for iface, a := range sums {
		bps := uint64(float64(a.bytes) / float64(a.n))
		totalBytes += bps
		result = append(result, ProtocolSample{Interface: iface, BytesPerSec: bps})
	}

	if totalBytes > 0 {
		for i := range result {
			result[i].Percent = float32(float64(result[i].BytesPerSec) / float64(totalBytes) * 100)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].BytesPerSec > result[j].BytesPerSec
	})

	return result
}
