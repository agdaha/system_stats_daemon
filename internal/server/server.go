package server

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	statspb "system_stats_deamon/api/stats"
	"system_stats_deamon/internal/collector/cpu"
	"system_stats_deamon/internal/collector/loadavg"
)

type Deps struct {
	LoadAvg *loadavg.Collector
	CPU     *cpu.Collector
}

type Server struct {
	statspb.UnimplementedStatsServiceServer
	deps Deps
}

func New(deps Deps) *Server {
	return &Server{deps: deps}
}

func (s *Server) GetStats(req *statspb.StatsRequest, stream grpc.ServerStreamingServer[statspb.Snapshot]) error {
	n := req.GetN()
	m := req.GetM()
	if n == 0 || m == 0 {
		return status.Error(codes.InvalidArgument, "N and M must be greater than 0")
	}

	interval := time.Duration(n) * time.Second
	window := time.Duration(m) * time.Second

	select {
	case <-time.After(window):
	case <-stream.Context().Done():
		return stream.Context().Err()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := stream.Send(s.buildSnapshot(window)); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func (s *Server) buildSnapshot(window time.Duration) *statspb.Snapshot {
	snap := &statspb.Snapshot{}

	if s.deps.LoadAvg != nil {
		if sample, ok := s.deps.LoadAvg.Average(window); ok {
			snap.LoadAverage = &statspb.LoadAverage{
				One:     sample.One,
				Five:    sample.Five,
				Fifteen: sample.Fifteen,
			}
		}
	}

	if s.deps.CPU != nil {
		if sample, ok := s.deps.CPU.Average(window); ok {
			snap.Cpu = &statspb.CPU{
				UserMode:   sample.User,
				SystemMode: sample.System,
				Idle:       sample.Idle,
			}
		}
	}

	return snap
}
