//go:build linux

package cpu

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type cpuStat struct {
	user, nice, system, idle, iowait, irq, softirq, steal uint64
}

func (st cpuStat) total() uint64 {
	return st.user + st.nice + st.system + st.idle + st.iowait + st.irq + st.softirq + st.steal
}

func (c *Collector) run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	prev, err := readCPUStat()
	if err != nil {
		return
	}

	for {
		select {
		case <-ticker.C:
			curr, err := readCPUStat()
			if err != nil {
				continue
			}
			if s, ok := computeDiff(prev, curr); ok {
				c.buf.Push(s)
			}
			prev = curr
		case <-ctx.Done():
			return
		}
	}
}

func readCPUStat() (cpuStat, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuStat{}, err
	}
	return parseCPUStat(string(data))
}

func parseCPUStat(content string) (cpuStat, error) {
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)

		if len(fields) < 5 {
			return cpuStat{}, fmt.Errorf("cpu: too few fields in /proc/stat line: %q", line)
		}

		vals := make([]uint64, 8)
		for i := range min(8, len(fields)-1) {
			v, err := strconv.ParseUint(fields[i+1], 10, 64)
			if err != nil {
				return cpuStat{}, fmt.Errorf("cpu: parse field %d: %w", i, err)
			}
			vals[i] = v
		}

		return cpuStat{
			user:    vals[0],
			nice:    vals[1],
			system:  vals[2],
			idle:    vals[3],
			iowait:  vals[4],
			irq:     vals[5],
			softirq: vals[6],
			steal:   vals[7],
		}, nil
	}

	return cpuStat{}, fmt.Errorf("cpu: no 'cpu' aggregate line found in /proc/stat")
}

func computeDiff(prev, curr cpuStat) (Sample, bool) {
	dTotal := float64(curr.total()) - float64(prev.total())
	if dTotal <= 0 {
		return Sample{}, false
	}

	scale := 100.0 / dTotal
	dUser := float64(curr.user+curr.nice) - float64(prev.user+prev.nice)
	dSystem := float64(curr.system+curr.irq+curr.softirq) - float64(prev.system+prev.irq+prev.softirq)
	dIdle := float64(curr.idle) - float64(prev.idle)

	return Sample{
		User:   float32(max(0.0, dUser) * scale),
		System: float32(max(0.0, dSystem) * scale),
		Idle:   float32(max(0.0, dIdle) * scale),
	}, true
}
