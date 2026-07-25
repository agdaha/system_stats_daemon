//go:build linux

package netsockets

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	protoTCP  = "tcp"
	protoTCP6 = "tcp6"
	protoUDP  = "udp"
	protoUDP6 = "udp6"
)

const collectInterval = 5 * time.Second

func (c *Collector) run(ctx context.Context) {
	ticker := time.NewTicker(collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			snap, err := collect()
			if err == nil {
				c.buf.Push(snap)
			}
		case <-ctx.Done():
			return
		}
	}
}

func collect() (socketSnapshot, error) {
	ssOut, err := exec.Command("ss", "-ta").Output()
	if err != nil {
		return socketSnapshot{}, fmt.Errorf("ss -ta: %w", err)
	}
	netstatOut, err := exec.Command("netstat", "-lntup").Output()
	if err != nil {
		return socketSnapshot{}, fmt.Errorf("netstat -lntup: %w", err)
	}

	return socketSnapshot{
		states:  parseSSTA(string(ssOut)),
		sockets: parseNetstat(string(netstatOut)),
	}, nil
}

func parseSSTA(content string) []StateSample {
	counts := make(map[string]uint32)
	for i, line := range strings.Split(content, "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		counts[fields[0]]++
	}

	states := make([]StateSample, 0, len(counts))
	for state, count := range counts {
		states = append(states, StateSample{State: state, Count: count})
	}
	sort.Slice(states, func(i, j int) bool {
		return states[i].Count > states[j].Count
	})
	return states
}

func parseNetstat(content string) []ListenSample {
	var sockets []ListenSample
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		proto := fields[0]
		switch proto {
		case protoTCP, protoTCP6:
			// TCP columns: Proto Recv-Q Send-Q Local Foreign State PID/Prog.
			if len(fields) < 7 || fields[5] != "LISTEN" {
				continue
			}
			port, err := extractPort(fields[3])
			if err != nil {
				continue
			}
			pid, cmd := parsePIDProg(fields[6])
			sockets = append(sockets, ListenSample{Protocol: proto, Port: port, PID: pid, Command: cmd})
		case protoUDP, protoUDP6:
			// UDP columns: Proto Recv-Q Send-Q Local Foreign PID/Prog (no State).
			if len(fields) < 6 {
				continue
			}
			port, err := extractPort(fields[3])
			if err != nil {
				continue
			}
			pid, cmd := parsePIDProg(fields[5])
			sockets = append(sockets, ListenSample{Protocol: proto, Port: port, PID: pid, Command: cmd})
		}
	}
	return sockets
}

func extractPort(addr string) (uint32, error) {
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return 0, fmt.Errorf("no port in %q", addr)
	}
	port, err := strconv.ParseUint(addr[i+1:], 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(port), nil
}

func parsePIDProg(s string) (pid uint32, command string) {
	if s == "-" {
		return 0, ""
	}
	parts := strings.SplitN(s, "/", 2)
	p, err := strconv.ParseUint(parts[0], 10, 32)
	if err == nil {
		pid = uint32(p)
	}
	if len(parts) > 1 {
		command = parts[1]
	}
	return pid, command
}
