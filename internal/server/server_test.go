package server

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	statspb "system_stats_deamon/api/stats"
	"system_stats_deamon/internal/collector/cpu"
	"system_stats_deamon/internal/collector/diskio"
	"system_stats_deamon/internal/collector/filesystem"
	"system_stats_deamon/internal/collector/loadavg"
	"system_stats_deamon/internal/collector/netsockets"
	"system_stats_deamon/internal/collector/nettraffic"
)

type mockStream struct {
	ctx     context.Context
	cancel  context.CancelFunc
	snaps   []*statspb.Snapshot
	maxRecv int
}

func newMockStream(parent context.Context, maxRecv int) *mockStream {
	ctx, cancel := context.WithCancel(parent)
	return &mockStream{ctx: ctx, cancel: cancel, maxRecv: maxRecv}
}

func (m *mockStream) Send(snap *statspb.Snapshot) error {
	m.snaps = append(m.snaps, snap)
	if m.maxRecv > 0 && len(m.snaps) >= m.maxRecv {
		m.cancel()
	}
	return nil
}

func (m *mockStream) Context() context.Context     { return m.ctx }
func (m *mockStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockStream) SendHeader(metadata.MD) error { return nil }
func (m *mockStream) SetTrailer(metadata.MD)       {}
func (m *mockStream) RecvMsg(any) error            { return nil }
func (m *mockStream) SendMsg(any) error            { return nil }

func newTestServer(deps Deps) *Server {
	return &Server{deps: deps, tickerDur: 10 * time.Millisecond, windowDur: 10 * time.Millisecond}
}

// --- mock collectors ---

type mockLoadAvg struct {
	sample loadavg.Sample
	ok     bool
}

func (m *mockLoadAvg) Average(_ time.Duration) (loadavg.Sample, bool) { return m.sample, m.ok }

type mockCPU struct {
	sample cpu.Sample
	ok     bool
}

func (m *mockCPU) Average(_ time.Duration) (cpu.Sample, bool) { return m.sample, m.ok }

type mockDiskIO struct {
	samples []diskio.Sample
}

func (m *mockDiskIO) Average(_ time.Duration) []diskio.Sample { return m.samples }

type mockFilesystem struct {
	samples []filesystem.Sample
}

func (m *mockFilesystem) Average(_ time.Duration) []filesystem.Sample { return m.samples }

type mockNetTraffic struct {
	protocols []nettraffic.ProtocolSample
	flows     []nettraffic.FlowSample
}

func (m *mockNetTraffic) Snapshot(_ time.Duration) ([]nettraffic.ProtocolSample, []nettraffic.FlowSample) {
	return m.protocols, m.flows
}

type mockNetSockets struct {
	sockets []netsockets.ListenSample
	states  []netsockets.StateSample
}

func (m *mockNetSockets) Snapshot(_ time.Duration) ([]netsockets.ListenSample, []netsockets.StateSample) {
	return m.sockets, m.states
}

// --- tests ---

func TestGetStats_InvalidArg(t *testing.T) {
	t.Parallel()
	srv := New(Deps{})
	stream := newMockStream(context.Background(), 0)

	if code := status.Code(srv.GetStats(&statspb.StatsRequest{N: 0, M: 1}, stream)); code != codes.InvalidArgument {
		t.Errorf("N=0: want InvalidArgument, got %s", code)
	}
	if code := status.Code(srv.GetStats(&statspb.StatsRequest{N: 1, M: 0}, stream)); code != codes.InvalidArgument {
		t.Errorf("M=0: want InvalidArgument, got %s", code)
	}
}

func TestGetStats_CancelledContext(t *testing.T) {
	t.Parallel()
	srv := newTestServer(Deps{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stream := newMockStream(ctx, 0)

	err := srv.GetStats(&statspb.StatsRequest{N: 1, M: 1}, stream)
	if err == nil {
		t.Error("expected error from cancelled context, got nil")
	}
	if len(stream.snaps) != 0 {
		t.Errorf("expected no snapshots before first send, got %d", len(stream.snaps))
	}
}

func TestGetStats_SendsSnapshots(t *testing.T) {
	t.Parallel()
	mock := &mockLoadAvg{sample: loadavg.Sample{One: 1.5, Five: 5.5, Fifteen: 15.5}, ok: true}
	srv := newTestServer(Deps{LoadAvg: mock})
	stream := newMockStream(context.Background(), 2)

	err := srv.GetStats(&statspb.StatsRequest{N: 1, M: 1}, stream)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("unexpected error: %v", err)
	}
	if len(stream.snaps) < 2 {
		t.Fatalf("want >=2 snapshots, got %d", len(stream.snaps))
	}
	if stream.snaps[0].GetLoadAverage().GetOne() != 1.5 {
		t.Errorf("first snapshot LoadAvg.One: want 1.5, got %v", stream.snaps[0].GetLoadAverage().GetOne())
	}
}

func TestGetStats_Concurrent(t *testing.T) {
	t.Parallel()
	mock := &mockLoadAvg{sample: loadavg.Sample{One: 1.0}, ok: true}
	srv := newTestServer(Deps{LoadAvg: mock})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := newMockStream(context.Background(), 1)
			_ = srv.GetStats(&statspb.StatsRequest{N: 1, M: 1}, s)
		}()
	}
	wg.Wait()
}

func TestBuildSnapshot_Empty(t *testing.T) {
	t.Parallel()
	srv := New(Deps{})
	snap := srv.buildSnapshot(time.Second)
	if snap.GetLoadAverage() != nil || snap.GetCpu() != nil ||
		len(snap.GetDiskIo()) > 0 || len(snap.GetFilesystems()) > 0 || snap.GetNetwork() != nil {
		t.Error("expected all-nil snapshot with empty Deps")
	}
}

func TestBuildSnapshot_LoadAvg(t *testing.T) {
	t.Parallel()
	mock := &mockLoadAvg{sample: loadavg.Sample{One: 1.0, Five: 5.0, Fifteen: 15.0}, ok: true}
	srv := New(Deps{LoadAvg: mock})
	la := srv.buildSnapshot(time.Second).GetLoadAverage()
	if la == nil {
		t.Fatal("expected non-nil LoadAverage")
	}
	if la.GetOne() != 1.0 || la.GetFive() != 5.0 || la.GetFifteen() != 15.0 {
		t.Errorf("LoadAverage: got {%.1f %.1f %.1f}", la.GetOne(), la.GetFive(), la.GetFifteen())
	}
}

func TestBuildSnapshot_LoadAvgNoData(t *testing.T) {
	t.Parallel()
	srv := New(Deps{LoadAvg: &mockLoadAvg{ok: false}})
	if srv.buildSnapshot(time.Second).GetLoadAverage() != nil {
		t.Error("expected nil LoadAverage when collector returns no data")
	}
}

func TestBuildSnapshot_CPU(t *testing.T) {
	t.Parallel()
	mock := &mockCPU{sample: cpu.Sample{User: 10.0, System: 5.0, Idle: 85.0}, ok: true}
	srv := New(Deps{CPU: mock})
	c := srv.buildSnapshot(time.Second).GetCpu()
	if c == nil {
		t.Fatal("expected non-nil CPU")
	}
	if c.GetUserMode() != 10.0 || c.GetSystemMode() != 5.0 || c.GetIdle() != 85.0 {
		t.Errorf("CPU: got {%.1f %.1f %.1f}", c.GetUserMode(), c.GetSystemMode(), c.GetIdle())
	}
}

func TestBuildSnapshot_DiskIO(t *testing.T) {
	t.Parallel()
	mock := &mockDiskIO{samples: []diskio.Sample{{Device: "sda", TPS: 10.5, KBps: 1024.0}}}
	srv := New(Deps{DiskIO: mock})
	disks := srv.buildSnapshot(time.Second).GetDiskIo()
	if len(disks) != 1 {
		t.Fatalf("want 1 disk entry, got %d", len(disks))
	}
	if disks[0].GetDevice() != "sda" || disks[0].GetTps() != 10.5 {
		t.Errorf("DiskIO[0]: unexpected %+v", disks[0])
	}
}

func TestBuildSnapshot_Filesystem(t *testing.T) {
	t.Parallel()
	mock := &mockFilesystem{samples: []filesystem.Sample{
		{Filesystem: "ext4", MountPoint: "/", UsedMB: 1000, UsedPercent: 50.0},
	}}
	srv := New(Deps{Filesystem: mock})
	fss := srv.buildSnapshot(time.Second).GetFilesystems()
	if len(fss) != 1 {
		t.Fatalf("want 1 filesystem entry, got %d", len(fss))
	}
	if fss[0].GetMountPoint() != "/" || fss[0].GetUsedMb() != 1000 {
		t.Errorf("Filesystem[0]: unexpected %+v", fss[0])
	}
}

func TestBuildSnapshot_Network(t *testing.T) {
	t.Parallel()
	nt := &mockNetTraffic{
		protocols: []nettraffic.ProtocolSample{{Interface: "eth0", BytesPerSec: 1000, Percent: 100.0}},
		flows: []nettraffic.FlowSample{
			{SrcAddr: "1.2.3.4", DstAddr: "5.6.7.8", Protocol: "tcp", Bps: 500.0},
		},
	}
	ns := &mockNetSockets{
		sockets: []netsockets.ListenSample{{Command: "nginx", PID: 1234, Protocol: "tcp", Port: 80}},
		states:  []netsockets.StateSample{{State: "LISTEN", Count: 5}},
	}
	net := New(Deps{NetTraffic: nt, NetSockets: ns}).buildSnapshot(time.Second).GetNetwork()
	if net == nil {
		t.Fatal("expected non-nil Network")
	}
	if len(net.GetProtocolTalkers()) != 1 || net.GetProtocolTalkers()[0].GetProtocol() != "eth0" {
		t.Errorf("ProtocolTalkers: unexpected %v", net.GetProtocolTalkers())
	}
	if len(net.GetListeningSockets()) != 1 || net.GetListeningSockets()[0].GetCommand() != "nginx" {
		t.Errorf("ListeningSockets: unexpected %v", net.GetListeningSockets())
	}
	if len(net.GetTcpStates()) != 1 || net.GetTcpStates()[0].GetState() != "LISTEN" {
		t.Errorf("TcpStates: unexpected %v", net.GetTcpStates())
	}
}
