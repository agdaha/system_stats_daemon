package server

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	statspb "system_stats_deamon/api/stats"
	"system_stats_deamon/internal/collector/cpu"
	"system_stats_deamon/internal/collector/diskio"
	"system_stats_deamon/internal/collector/filesystem"
	"system_stats_deamon/internal/collector/loadavg"
	"system_stats_deamon/internal/collector/netsockets"
	"system_stats_deamon/internal/collector/nettraffic"
)

type (
	loadAvgProvider interface {
		Average(time.Duration) (loadavg.Sample, bool)
	}
	cpuProvider interface {
		Average(time.Duration) (cpu.Sample, bool)
	}
	diskIOProvider interface {
		Average(time.Duration) []diskio.Sample
	}
	filesystemProvider interface {
		Average(time.Duration) []filesystem.Sample
	}
	netTrafficProvider interface {
		Snapshot(time.Duration) ([]nettraffic.ProtocolSample, []nettraffic.FlowSample)
	}
	netSocketsProvider interface {
		Snapshot(time.Duration) ([]netsockets.ListenSample, []netsockets.StateSample)
	}
)
type Deps struct {
	LoadAvg    loadAvgProvider
	CPU        cpuProvider
	DiskIO     diskIOProvider
	Filesystem filesystemProvider
	NetTraffic netTrafficProvider
	NetSockets netSocketsProvider
}
type Server struct {
	statspb.UnimplementedStatsServiceServer
	deps      Deps
	tickerDur time.Duration
	windowDur time.Duration
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

	interval := s.tickerDur
	if interval == 0 {
		interval = time.Duration(n) * time.Second
	}
	window := s.windowDur
	if window == 0 {
		window = time.Duration(m) * time.Second
	}

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

	if s.deps.DiskIO != nil {
		for _, sample := range s.deps.DiskIO.Average(window) {
			snap.DiskIo = append(snap.DiskIo, &statspb.DiskIO{
				Device: sample.Device,
				Tps:    sample.TPS,
				Kbps:   sample.KBps,
			})
		}
	}

	if s.deps.Filesystem != nil {
		for _, sample := range s.deps.Filesystem.Average(window) {
			snap.Filesystems = append(snap.Filesystems, &statspb.Filesystem{
				Filesystem:        sample.Filesystem,
				MountPoint:        sample.MountPoint,
				UsedMb:            sample.UsedMB,
				UsedPercent:       sample.UsedPercent,
				UsedInodes:        sample.UsedInodes,
				UsedInodesPercent: sample.UsedInodesPercent,
			})
		}
	}

	if s.deps.NetTraffic != nil || s.deps.NetSockets != nil {
		snap.Network = s.buildNetwork(window)
	}

	return snap
}

func (s *Server) buildNetwork(window time.Duration) *statspb.NetworkStats {
	net := &statspb.NetworkStats{}

	if s.deps.NetTraffic != nil {
		protocols, flows := s.deps.NetTraffic.Snapshot(window)
		for _, p := range protocols {
			net.ProtocolTalkers = append(net.ProtocolTalkers, &statspb.ProtocolTalker{
				Protocol: p.Interface,
				Bytes:    p.BytesPerSec,
				Percent:  p.Percent,
			})
		}
		for _, f := range flows {
			net.TrafficTalkers = append(net.TrafficTalkers, &statspb.TrafficTalker{
				SrcAddr:  f.SrcAddr,
				DstAddr:  f.DstAddr,
				Protocol: f.Protocol,
				Bps:      f.Bps,
			})
		}
	}

	if s.deps.NetSockets != nil {
		sockets, states := s.deps.NetSockets.Snapshot(window)
		for _, sock := range sockets {
			net.ListeningSockets = append(net.ListeningSockets, &statspb.ListeningSocket{
				Command:  sock.Command,
				Pid:      sock.PID,
				Protocol: sock.Protocol,
				Port:     sock.Port,
			})
		}
		for _, st := range states {
			net.TcpStates = append(net.TcpStates, &statspb.TCPState{
				State: st.State,
				Count: st.Count,
			})
		}
	}

	return net
}
