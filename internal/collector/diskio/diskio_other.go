//go:build !linux

package diskio

import "context"

func (c *Collector) run(_ context.Context) {}
