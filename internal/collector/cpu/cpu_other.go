//go:build !linux && !darwin

package cpu

import "context"

func (c *Collector) run(_ context.Context) {}
