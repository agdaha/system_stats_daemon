//go:build linux

package nettraffic

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type netStat struct {
	iface   string
	rxBytes uint64
	txBytes uint64
}

func (c *Collector) runProto(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	prev, err := readNetDev()
	if err != nil {
		return
	}

	for {
		select {
		case <-ticker.C:
			curr, err := readNetDev()
			if err != nil {
				continue
			}
			if samples := diffNetDev(prev, curr); len(samples) > 0 {
				c.protoBuf.Push(samples)
			}
			prev = curr
		case <-ctx.Done():
			return
		}
	}
}

func readNetDev() ([]netStat, error) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	return parseNetDev(string(data))
}

func parseNetDev(content string) ([]netStat, error) {
	lines := strings.Split(content, "\n")
	stats := make([]netStat, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 10 || !strings.HasSuffix(fields[0], ":") {
			continue
		}
		iface := strings.TrimSuffix(fields[0], ":")
		rxBytes, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("net/dev: rx bytes for %s: %w", iface, err)
		}
		txBytes, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("net/dev: tx bytes for %s: %w", iface, err)
		}
		stats = append(stats, netStat{iface: iface, rxBytes: rxBytes, txBytes: txBytes})
	}
	return stats, nil
}

func diffNetDev(prev, curr []netStat) []ProtocolSample {
	prevMap := make(map[string]netStat, len(prev))
	for _, s := range prev {
		prevMap[s.iface] = s
	}

	samples := make([]ProtocolSample, 0, len(curr))
	for _, c := range curr {
		p, ok := prevMap[c.iface]
		if !ok {
			continue
		}
		dRx := float64(c.rxBytes) - float64(p.rxBytes)
		dTx := float64(c.txBytes) - float64(p.txBytes)
		total := dRx + dTx
		if total < 0 {
			total = 0
		}
		samples = append(samples, ProtocolSample{
			Interface:   c.iface,
			BytesPerSec: uint64(total),
		})
	}
	return samples
}

const flowCaptureDur = 5 * time.Second

func (c *Collector) runFlows(ctx context.Context) {
	for ctx.Err() == nil {
		flows := captureFlows(ctx, flowCaptureDur)
		if len(flows) > 0 {
			c.flowBuf.Push(flows)
		}
	}
}

func captureFlows(ctx context.Context, dur time.Duration) []FlowSample {
	captureCtx, cancel := context.WithTimeout(ctx, dur)
	defer cancel()

	var buf bytes.Buffer
	cmd := exec.CommandContext(captureCtx, "tcpdump", "-nt", "-l", "-q", "-i", "any")
	cmd.Stdout = &buf

	if err := cmd.Start(); err != nil {
		return nil // tcpdump unavailable or permission denied
	}
	_ = cmd.Wait()

	if ctx.Err() != nil {
		return nil
	}
	return parseFlows(buf.String(), dur)
}

func parseFlows(content string, dur time.Duration) []FlowSample {
	type flowKey struct{ src, dst, proto string }
	accumMap := make(map[flowKey]int64)

	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 || fields[0] != "IP" || fields[2] != ">" {
			continue
		}
		src := stripPort(fields[1])
		dst := stripPort(strings.TrimSuffix(fields[3], ":"))
		proto := strings.ToLower(strings.TrimSuffix(fields[4], ","))
		last := fields[len(fields)-1]
		length, err := strconv.ParseInt(last, 10, 64)
		if err != nil || length <= 0 {
			continue
		}
		key := flowKey{src, dst, proto}
		accumMap[key] += length
	}

	secs := dur.Seconds()
	result := make([]FlowSample, 0, len(accumMap))
	for k, b := range accumMap {
		result = append(result, FlowSample{
			SrcAddr:  k.src,
			DstAddr:  k.dst,
			Protocol: k.proto,
			Bps:      float32(float64(b) / secs),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Bps > result[j].Bps
	})
	return result
}

func stripPort(addr string) string {
	i := strings.LastIndex(addr, ".")
	if i < 0 {
		return addr
	}
	return addr[:i]
}
