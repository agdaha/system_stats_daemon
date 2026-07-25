package nettraffic

import (
	"bytes"
	"context"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

const flowCaptureDur = 5 * time.Second

// captureFlows runs tcpdump for dur and returns per-flow BPS samples.
// Returns nil if tcpdump is unavailable or permission is denied.
func captureFlows(ctx context.Context, dur time.Duration) []FlowSample {
	captureCtx, cancel := context.WithTimeout(ctx, dur)
	defer cancel()

	var buf bytes.Buffer
	cmd := exec.CommandContext(captureCtx, "tcpdump", "-nt", "-l", "-q", "-i", "any")
	cmd.Stdout = &buf

	if err := cmd.Start(); err != nil {
		return nil
	}
	_ = cmd.Wait()

	if ctx.Err() != nil {
		return nil
	}
	return parseFlows(buf.String(), dur)
}

// parseFlows extracts per-flow BPS from tcpdump -nt -q output.
// Expected line format: "IP src.port > dst.port: proto ... length N".
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

// stripPort removes the last dot-segment from a tcpdump IPv4 "addr.port" string.
// E.g. "192.168.1.1.12345" → "192.168.1.1".
func stripPort(addr string) string {
	i := strings.LastIndex(addr, ".")
	if i < 0 {
		return addr
	}
	return addr[:i]
}
