//go:build darwin

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
	tcpOut, err := exec.Command("netstat", "-an", "-p", "tcp").Output()
	if err != nil {
		return socketSnapshot{}, fmt.Errorf("netstat -an -p tcp: %w", err)
	}
	udpOut, err := exec.Command("netstat", "-an", "-p", "udp").Output()
	if err != nil {
		return socketSnapshot{}, fmt.Errorf("netstat -an -p udp: %w", err)
	}
	return socketSnapshot{
		states:  parseTCPStates(string(tcpOut)),
		sockets: parseListeningSockets(string(tcpOut), string(udpOut)),
	}, nil
}

// parseTCPStates counts TCP connection states from "netstat -an -p tcp" output.
// The state is the last field of each data line.
// Example: "tcp4  0  0  127.0.0.1.8080  *.*  LISTEN"
func parseTCPStates(content string) []StateSample {
	counts := make(map[string]uint32)
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		if !strings.HasPrefix(fields[0], "tcp") {
			continue
		}
		state := fields[len(fields)-1]
		if state == "(state)" {
			continue
		}
		counts[state]++
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

// parseListeningSockets extracts listening TCP and bound UDP sockets.
// On Darwin, PID/command are not available from netstat without root; those fields are zero/"".
func parseListeningSockets(tcpOut, udpOut string) []ListenSample {
	var sockets []ListenSample
	sockets = append(sockets, parseTCPListeners(tcpOut)...)
	sockets = append(sockets, parseUDPListeners(udpOut)...)
	return sockets
}

func parseTCPListeners(content string) []ListenSample {
	var sockets []ListenSample
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		if !strings.HasPrefix(fields[0], "tcp") {
			continue
		}
		if fields[len(fields)-1] != "LISTEN" {
			continue
		}
		port, err := extractPort(fields[3])
		if err != nil {
			continue
		}
		sockets = append(sockets, ListenSample{Protocol: fields[0], Port: port})
	}
	return sockets
}

func parseUDPListeners(content string) []ListenSample {
	var sockets []ListenSample
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		if !strings.HasPrefix(fields[0], "udp") {
			continue
		}
		localAddr := fields[3]
		if localAddr == "*.*" {
			continue
		}
		port, err := extractPort(localAddr)
		if err != nil {
			continue
		}
		sockets = append(sockets, ListenSample{Protocol: fields[0], Port: port})
	}
	return sockets
}

// extractPort parses the port from Darwin's "addr.port" notation.
// E.g. "127.0.0.1.8080" → 8080, "*.8080" → 8080.
func extractPort(addr string) (uint32, error) {
	i := strings.LastIndex(addr, ".")
	if i < 0 {
		return 0, fmt.Errorf("no port in %q", addr)
	}
	portStr := addr[i+1:]
	if portStr == "*" || portStr == "" {
		return 0, fmt.Errorf("wildcard port in %q", addr)
	}
	port, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(port), nil
}
