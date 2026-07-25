//go:build darwin

package nettraffic

import (
	"context"
	"fmt"
	"os/exec"
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

	prev, err := readNetStats()
	if err != nil {
		return
	}

	for {
		select {
		case <-ticker.C:
			curr, err := readNetStats()
			if err != nil {
				continue
			}
			if samples := diffNetStats(prev, curr); len(samples) > 0 {
				c.protoBuf.Push(samples)
			}
			prev = curr
		case <-ctx.Done():
			return
		}
	}
}

func readNetStats() ([]netStat, error) {
	out, err := exec.Command("netstat", "-ib").Output()
	if err != nil {
		return nil, fmt.Errorf("netstat -ib: %w", err)
	}
	return parseNetstatIB(string(out))
}

// parseNetstatIB parses "netstat -ib" output on macOS.
// Only <Link#N> rows carry cumulative byte counters.
// Columns: Name Mtu Network Address Ipkts Ierrs Ibytes Opkts Oerrs Obytes Coll.
func parseNetstatIB(content string) ([]netStat, error) {
	var stats []netStat
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		if !strings.HasPrefix(fields[2], "<Link") {
			continue
		}
		iface := fields[0]
		rxBytes, err := strconv.ParseUint(fields[6], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("netstat -ib: rx bytes for %s: %w", iface, err)
		}
		txBytes, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("netstat -ib: tx bytes for %s: %w", iface, err)
		}
		stats = append(stats, netStat{iface: iface, rxBytes: rxBytes, txBytes: txBytes})
	}
	return stats, nil
}

func diffNetStats(prev, curr []netStat) []ProtocolSample {
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
		samples = append(samples, ProtocolSample{Interface: c.iface, BytesPerSec: uint64(total)})
	}
	return samples
}

func (c *Collector) runFlows(ctx context.Context) {
	for ctx.Err() == nil {
		flows := captureFlows(ctx, flowCaptureDur)
		if len(flows) > 0 {
			c.flowBuf.Push(flows)
		}
	}
}
