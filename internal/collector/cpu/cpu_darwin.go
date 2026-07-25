//go:build darwin

package cpu

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func (c *Collector) run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if s, err := readCPU(); err == nil {
				c.buf.Push(s)
			}
		case <-ctx.Done():
			return
		}
	}
}

func readCPU() (Sample, error) {
	out, err := exec.Command("top", "-l", "1", "-n", "0").Output()
	if err != nil {
		return Sample{}, err
	}
	return parseCPU(string(out))
}

func parseCPU(content string) (Sample, error) {
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "CPU usage:") {
			continue
		}
		return parseCPULine(line)
	}
	return Sample{}, fmt.Errorf("cpu: no 'CPU usage:' line found in top output")
}

func parseCPULine(line string) (Sample, error) {
	fields := strings.Fields(line)
	// Expected: "CPU", "usage:", "3.77%", "user,", "11.32%", "sys,", "84.90%", "idle"
	if len(fields) < 8 {
		return Sample{}, fmt.Errorf("cpu: unexpected CPU usage line: %q", line)
	}

	user, err := parsePct(fields[2])
	if err != nil {
		return Sample{}, fmt.Errorf("cpu: parse user: %w", err)
	}
	sys, err := parsePct(fields[4])
	if err != nil {
		return Sample{}, fmt.Errorf("cpu: parse sys: %w", err)
	}
	idle, err := parsePct(fields[6])
	if err != nil {
		return Sample{}, fmt.Errorf("cpu: parse idle: %w", err)
	}

	return Sample{User: user, System: sys, Idle: idle}, nil
}

func parsePct(s string) (float32, error) {
	s = strings.TrimSuffix(strings.TrimSuffix(s, ","), "%")
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, err
	}
	return float32(v), nil
}
