//go:build !linux && !darwin

package netsockets

import "context"

func (c *Collector) run(_ context.Context) {}
