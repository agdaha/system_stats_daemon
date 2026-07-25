//go:build integration && linux

package integration

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	statspb "system_stats_deamon/api/stats"
	"system_stats_deamon/internal/collector/cpu"
	"system_stats_deamon/internal/collector/diskio"
	"system_stats_deamon/internal/collector/filesystem"
	"system_stats_deamon/internal/collector/loadavg"
	"system_stats_deamon/internal/server"
)

const (
	testN uint32 = 1 // snapshot interval, seconds
	testM uint32 = 1 // averaging window, seconds
)

func startServer(t *testing.T) string {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	la := loadavg.New()
	la.Start(ctx)
	cp := cpu.New()
	cp.Start(ctx)
	di := diskio.New()
	di.Start(ctx)
	fs := filesystem.New()
	fs.Start(ctx)

	deps := server.Deps{LoadAvg: la, CPU: cp, DiskIO: di, Filesystem: fs}
	grpcSrv := grpc.NewServer()
	statspb.RegisterStatsServiceServer(grpcSrv, server.New(deps))

	go func() { _ = grpcSrv.Serve(lis) }()
	t.Cleanup(grpcSrv.GracefulStop)

	return lis.Addr().String()
}

func newConn(t *testing.T, addr string) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestIntegration_SnapshotsArrive(t *testing.T) {
	client := statspb.NewStatsServiceClient(newConn(t, startServer(t)))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.GetStats(ctx, &statspb.StatsRequest{N: testN, M: testM})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}

	snap, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}

	if snap.GetLoadAverage() == nil {
		t.Error("expected non-nil LoadAverage in first snapshot")
	}
	if snap.GetCpu() == nil {
		t.Error("expected non-nil CPU in first snapshot")
	}

	if len(snap.GetFilesystems()) == 0 {
		t.Log("Filesystems empty (collector interval > window, OK)")
	}
	if len(snap.GetDiskIo()) == 0 {
		t.Log("DiskIO empty (no physical disks or collector interval > window)")
	}
}

func TestIntegration_FilesystemData(t *testing.T) {
	client := statspb.NewStatsServiceClient(newConn(t, startServer(t)))

	// M=6s ensures the filesystem collector (5s interval) has one sample.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	stream, err := client.GetStats(ctx, &statspb.StatsRequest{N: 2, M: 6})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}

	snap, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}

	if len(snap.GetFilesystems()) == 0 {
		t.Error("expected at least one filesystem entry with M=6s window")
	}
	for _, fs := range snap.GetFilesystems() {
		if fs.GetMountPoint() == "" {
			t.Error("filesystem entry has empty mount point")
		}
	}
}

func TestIntegration_SnapshotValues(t *testing.T) {
	client := statspb.NewStatsServiceClient(newConn(t, startServer(t)))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.GetStats(ctx, &statspb.StatsRequest{N: testN, M: testM})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}

	snap, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}

	if la := snap.GetLoadAverage(); la != nil {
		for _, v := range []float32{la.GetOne(), la.GetFive(), la.GetFifteen()} {
			if v < 0 {
				t.Errorf("LoadAverage has negative component: %.2f", v)
			}
		}
	}

	if c := snap.GetCpu(); c != nil {
		for _, pct := range []float32{c.GetUserMode(), c.GetSystemMode(), c.GetIdle()} {
			if pct < 0 || pct > 100 {
				t.Errorf("CPU percentage out of [0, 100]: %.1f%%", pct)
			}
		}
	}

	for _, fs := range snap.GetFilesystems() {
		if fs.GetMountPoint() == "" {
			t.Error("filesystem entry has empty mount point")
		}
		if fs.GetUsedPercent() < 0 || fs.GetUsedPercent() > 100 {
			t.Errorf("filesystem %s: UsedPercent=%.1f%% is out of range",
				fs.GetMountPoint(), fs.GetUsedPercent())
		}
	}
}

func TestIntegration_StreamsMultipleSnapshots(t *testing.T) {
	client := statspb.NewStatsServiceClient(newConn(t, startServer(t)))

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	stream, err := client.GetStats(ctx, &statspb.StatsRequest{N: testN, M: testM})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}

	for i := 0; i < 2; i++ {
		if _, err := stream.Recv(); err != nil {
			t.Fatalf("Recv snapshot %d: %v", i+1, err)
		}
	}
}

func TestIntegration_MultipleClients(t *testing.T) {
	addr := startServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const nClients = 3
	errs := make(chan error, nClients)

	for i := 0; i < nClients; i++ {
		go func() {
			conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				errs <- fmt.Errorf("dial: %w", err)
				return
			}
			defer func() { _ = conn.Close() }()

			stream, err := statspb.NewStatsServiceClient(conn).GetStats(
				ctx, &statspb.StatsRequest{N: testN, M: testM},
			)
			if err != nil {
				errs <- fmt.Errorf("GetStats: %w", err)
				return
			}
			_, err = stream.Recv()
			errs <- err
		}()
	}

	for i := 0; i < nClients; i++ {
		if err := <-errs; err != nil {
			t.Errorf("client %d error: %v", i+1, err)
		}
	}
}

func TestIntegration_InvalidArgs(t *testing.T) {
	client := statspb.NewStatsServiceClient(newConn(t, startServer(t)))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.GetStats(ctx, &statspb.StatsRequest{N: 0, M: 1})
	if err != nil {
		return // error returned immediately, as expected
	}
	if _, err = stream.Recv(); err == nil {
		t.Error("expected error for N=0, got nil")
	}
}

func TestIntegration_ClientDisconnect(t *testing.T) {
	client := statspb.NewStatsServiceClient(newConn(t, startServer(t)))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.GetStats(ctx, &statspb.StatsRequest{N: testN, M: testM})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}

	if _, err = stream.Recv(); err != nil {
		t.Fatalf("Recv first snapshot: %v", err)
	}

	cancel() // disconnect after first snapshot

	_, err = stream.Recv()
	if err == nil {
		t.Error("expected error after client disconnect, got nil")
	}
}
