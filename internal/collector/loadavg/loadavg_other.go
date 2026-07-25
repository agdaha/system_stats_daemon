//go:build !linux && !darwin

package loadavg

import "context"

func (c *Collector) run(_ context.Context) {}
