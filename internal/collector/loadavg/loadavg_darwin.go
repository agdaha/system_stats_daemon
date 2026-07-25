//go:build darwin

package loadavg

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
			if s, err := readLoadAvg(); err == nil {
				c.buf.Push(s)
			}
		case <-ctx.Done():
			return
		}
	}
}

func readLoadAvg() (Sample, error) {
	out, err := exec.Command("sysctl", "-n", "vm.loadavg").Output()
	if err != nil {
		return Sample{}, err
	}
	return parseLoadAvg(string(out))
}

// parseLoadAvg parses "sysctl -n vm.loadavg" output: "{ 1.23 0.45 0.12 }".
func parseLoadAvg(content string) (Sample, error) {
	content = strings.Trim(strings.TrimSpace(content), "{}")
	fields := strings.Fields(content)
	if len(fields) < 3 {
		return Sample{}, fmt.Errorf("loadavg: expected at least 3 fields, got %d", len(fields))
	}

	one, err := strconv.ParseFloat(fields[0], 32)
	if err != nil {
		return Sample{}, fmt.Errorf("loadavg: parse 1m: %w", err)
	}
	five, err := strconv.ParseFloat(fields[1], 32)
	if err != nil {
		return Sample{}, fmt.Errorf("loadavg: parse 5m: %w", err)
	}
	fifteen, err := strconv.ParseFloat(fields[2], 32)
	if err != nil {
		return Sample{}, fmt.Errorf("loadavg: parse 15m: %w", err)
	}

	return Sample{One: float32(one), Five: float32(five), Fifteen: float32(fifteen)}, nil
}
