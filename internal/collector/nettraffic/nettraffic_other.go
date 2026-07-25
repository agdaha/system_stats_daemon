//go:build !linux && !darwin

package nettraffic

import "context"

func (c *Collector) runProto(_ context.Context) {}
func (c *Collector) runFlows(_ context.Context) {}
