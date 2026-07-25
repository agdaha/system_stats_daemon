//go:build darwin

package diskio

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func (c *Collector) run(ctx context.Context) {
	for ctx.Err() == nil {
		samples, err := collectDiskIO()
		if err == nil && len(samples) > 0 {
			c.buf.Push(samples)
		}
	}
}

func collectDiskIO() ([]Sample, error) {
	out, err := exec.Command("iostat", "-d", "-w", "1", "-c", "2").Output()
	if err != nil {
		return nil, fmt.Errorf("iostat: %w", err)
	}
	return parseIOStat(string(out))
}

func parseIOStat(content string) ([]Sample, error) {
	var nonEmpty []string
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) != "" {
			nonEmpty = append(nonEmpty, line)
		}
	}
	// Need: device header + column header + 2 data rows = 4 lines minimum.
	if len(nonEmpty) < 4 {
		return nil, fmt.Errorf("iostat: too few non-empty lines (%d)", len(nonEmpty))
	}

	devices := strings.Fields(nonEmpty[0])
	if len(devices) == 0 {
		return nil, fmt.Errorf("iostat: no devices in header line")
	}

	// The last non-empty line is the second (1-second interval) sample.
	return parseDataLine(devices, nonEmpty[len(nonEmpty)-1])
}

func parseDataLine(devices []string, line string) ([]Sample, error) {
	fields := strings.Fields(line)
	if len(fields) < len(devices)*3 {
		return nil, fmt.Errorf("iostat: want %d fields, got %d in %q", len(devices)*3, len(fields), line)
	}

	samples := make([]Sample, 0, len(devices))
	for i, dev := range devices {
		tpsStr := fields[i*3+1]
		mbsStr := fields[i*3+2]

		tps, err := strconv.ParseFloat(tpsStr, 32)
		if err != nil {
			return nil, fmt.Errorf("iostat: parse tps for %s: %w", dev, err)
		}
		mbs, err := strconv.ParseFloat(mbsStr, 32)
		if err != nil {
			return nil, fmt.Errorf("iostat: parse MB/s for %s: %w", dev, err)
		}
		samples = append(samples, Sample{
			Device: dev,
			TPS:    float32(tps),
			KBps:   float32(mbs * 1024), // MB/s → KB/s
		})
	}
	return samples, nil
}
