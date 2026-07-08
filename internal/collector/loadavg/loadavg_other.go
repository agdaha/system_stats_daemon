//go:build !linux

package loadavg

import "context"

func (c *Collector) run(_ context.Context) {}
