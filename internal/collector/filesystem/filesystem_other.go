//go:build !linux

package filesystem

import "context"

func (c *Collector) run(_ context.Context) {}
