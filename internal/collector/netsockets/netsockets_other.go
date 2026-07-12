//go:build !linux

package netsockets

import "context"

func (c *Collector) run(_ context.Context) {}
