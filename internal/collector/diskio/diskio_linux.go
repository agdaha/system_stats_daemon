//go:build linux

package diskio

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type diskStat struct {
	device          string
	readsCompleted  uint64
	sectorsRead     uint64
	writesCompleted uint64
	sectorsWritten  uint64
}

func (c *Collector) run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	prev, err := readDiskStats()
	if err != nil {
		return
	}

	for {
		select {
		case <-ticker.C:
			curr, err := readDiskStats()
			if err != nil {
				continue
			}
			if samples := computeDiff(prev, curr); len(samples) > 0 {
				c.buf.Push(samples)
			}
			prev = curr
		case <-ctx.Done():
			return
		}
	}
}

func readDiskStats() ([]diskStat, error) {
	data, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		return nil, err
	}
	return parseDiskStats(string(data))
}

func parseDiskStats(content string) ([]diskStat, error) {
	lines := strings.Split(content, "\n")
	stats := make([]diskStat, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}
		device := fields[2]
		if strings.HasPrefix(device, "ram") || strings.HasPrefix(device, "loop") {
			continue
		}
		readsCompleted, err := strconv.ParseUint(fields[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("diskstats: reads for %s: %w", device, err)
		}
		sectorsRead, err := strconv.ParseUint(fields[5], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("diskstats: sectors read for %s: %w", device, err)
		}
		writesCompleted, err := strconv.ParseUint(fields[7], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("diskstats: writes for %s: %w", device, err)
		}
		sectorsWritten, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("diskstats: sectors written for %s: %w", device, err)
		}
		stats = append(stats, diskStat{
			device:          device,
			readsCompleted:  readsCompleted,
			sectorsRead:     sectorsRead,
			writesCompleted: writesCompleted,
			sectorsWritten:  sectorsWritten,
		})
	}
	return stats, nil
}

func computeDiff(prev, curr []diskStat) []Sample {
	prevMap := make(map[string]diskStat, len(prev))
	for _, s := range prev {
		prevMap[s.device] = s
	}

	samples := make([]Sample, 0, len(curr))
	for _, c := range curr {
		p, ok := prevMap[c.device]
		if !ok {
			continue
		}
		dOps := float64(c.readsCompleted) + float64(c.writesCompleted) -
			float64(p.readsCompleted) - float64(p.writesCompleted)
		dSectors := float64(c.sectorsRead) + float64(c.sectorsWritten) -
			float64(p.sectorsRead) - float64(p.sectorsWritten)
		if dOps < 0 {
			dOps = 0
		}
		if dSectors < 0 {
			dSectors = 0
		}
		samples = append(samples, Sample{
			Device: c.device,
			TPS:    float32(dOps),
			KBps:   float32(dSectors * 0.5), // 1 sector = 512 B = 0.5 KB
		})
	}
	return samples
}
