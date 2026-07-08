//go:build !linux

package cpu

import "context"

func (c *Collector) run(_ context.Context) {}
