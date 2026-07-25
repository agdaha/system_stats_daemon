//go:build !linux && !darwin

package filesystem

import "context"

func (c *Collector) run(_ context.Context) {}
