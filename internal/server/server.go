package server

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	statspb "system_stats_deamon/api/stats"
)

type Server struct {
	statspb.UnimplementedStatsServiceServer
}

func New() *Server {
	return &Server{}
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
			if err := stream.Send(&statspb.Snapshot{}); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}
