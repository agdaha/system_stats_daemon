//go:build linux

package filesystem

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const collectInterval = 5 * time.Second

func (c *Collector) run(ctx context.Context) {
	ticker := time.NewTicker(collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			samples, err := collect()
			if err == nil && len(samples) > 0 {
				c.buf.Push(samples)
			}
		case <-ctx.Done():
			return
		}
	}
}

func collect() ([]Sample, error) {
	spaceOut, err := runDF("-k")
	if err != nil {
		return nil, fmt.Errorf("df -k: %w", err)
	}
	inodeOut, err := runDF("-i")
	if err != nil {
		return nil, fmt.Errorf("df -i: %w", err)
	}

	space, err := parseDFk(spaceOut)
	if err != nil {
		return nil, err
	}
	inodes, err := parseDFi(inodeOut)
	if err != nil {
		return nil, err
	}

	return merge(space, inodes), nil
}

func runDF(flag string) (string, error) {
	out, err := exec.Command("df", "-P", flag).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

type dfkEntry struct {
	filesystem  string
	usedKB      uint64
	usedPercent float32
	mountPoint  string
}

func parseDFk(content string) ([]dfkEntry, error) {
	lines := strings.Split(content, "\n")
	entries := make([]dfkEntry, 0, len(lines))
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		usedKB, err := strconv.ParseUint(fields[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("df -k: parse used for %s: %w", fields[0], err)
		}
		pct, err := parsePercent(fields[4])
		if err != nil {
			return nil, fmt.Errorf("df -k: parse percent for %s: %w", fields[0], err)
		}
		entries = append(entries, dfkEntry{
			filesystem:  fields[0],
			usedKB:      usedKB,
			usedPercent: pct,
			mountPoint:  fields[5],
		})
	}
	return entries, nil
}

type dfiEntry struct {
	mountPoint        string
	usedInodes        uint64
	usedInodesPercent float32
}

func parseDFi(content string) ([]dfiEntry, error) {
	lines := strings.Split(content, "\n")
	entries := make([]dfiEntry, 0, len(lines))
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		usedInodes, err := strconv.ParseUint(fields[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("df -i: parse used inodes for %s: %w", fields[0], err)
		}
		pct, err := parsePercent(fields[4])
		if err != nil {
			return nil, fmt.Errorf("df -i: parse inode percent for %s: %w", fields[0], err)
		}
		entries = append(entries, dfiEntry{
			mountPoint:        fields[5],
			usedInodes:        usedInodes,
			usedInodesPercent: pct,
		})
	}
	return entries, nil
}

func parsePercent(s string) (float32, error) {
	s = strings.TrimSuffix(s, "%")
	if s == "-" {
		return 0, nil
	}
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, err
	}
	return float32(v), nil
}

func merge(space []dfkEntry, inodes []dfiEntry) []Sample {
	inodeMap := make(map[string]dfiEntry, len(inodes))
	for _, e := range inodes {
		inodeMap[e.mountPoint] = e
	}

	samples := make([]Sample, 0, len(space))
	for _, s := range space {
		inode := inodeMap[s.mountPoint]
		samples = append(samples, Sample{
			Filesystem:        s.filesystem,
			MountPoint:        s.mountPoint,
			UsedMB:            s.usedKB / 1024,
			UsedPercent:       s.usedPercent,
			UsedInodes:        inode.usedInodes,
			UsedInodesPercent: inode.usedInodesPercent,
		})
	}
	return samples
}
