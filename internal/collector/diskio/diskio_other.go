//go:build !linux && !darwin

package diskio

import "context"

func (c *Collector) run(_ context.Context) {}
