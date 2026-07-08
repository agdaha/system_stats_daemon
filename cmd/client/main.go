package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	statspb "system_stats_deamon/api/stats"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "daemon address host:port")
	n := flag.Uint("n", 5, "snapshot interval in seconds")
	m := flag.Uint("m", 15, "averaging window in seconds")
	flag.Parse()

	if *n == 0 || *m == 0 {
		log.Fatal("--n and --m must be greater than 0")
	}

	if err := run(*addr, uint32(*n), uint32(*m)); err != nil { //nolint:gosec // n,m validated >0 and reasonable
		log.Fatal(err)
	}
}

func run(addr string, n, m uint32) error {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	stream, err := statspb.NewStatsServiceClient(conn).GetStats(ctx, &statspb.StatsRequest{N: n, M: m})
	if err != nil {
		return fmt.Errorf("get stats: %w", err)
	}

	fmt.Printf("connected to %s  N=%ds  M=%ds\nwaiting %ds for first snapshot...\n", addr, n, m, m)

	for {
		snap, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("recv: %w", err)
		}
		printSnapshot(snap, addr, n, m)
	}
}

const lineWidth = 44

func printSnapshot(snap *statspb.Snapshot, addr string, n, m uint32) {
	fmt.Print("\033[2J\033[H") // clear screen, move cursor to top
	fmt.Printf("sysmon  %s  N=%ds M=%ds  %s\n",
		addr, n, m, time.Now().Format("15:04:05"))
	fmt.Println(strings.Repeat("─", lineWidth))

	if la := snap.GetLoadAverage(); la != nil {
		fmt.Println("LOAD AVERAGE")
		fmt.Printf("   1 min  %8.2f\n", la.GetOne())
		fmt.Printf("   5 min  %8.2f\n", la.GetFive())
		fmt.Printf("  15 min  %8.2f\n", la.GetFifteen())
	}

	if cpu := snap.GetCpu(); cpu != nil {
		fmt.Println("CPU")
		fmt.Printf("   user   %7.1f%%\n", cpu.GetUserMode())
		fmt.Printf("   system %7.1f%%\n", cpu.GetSystemMode())
		fmt.Printf("   idle   %7.1f%%\n", cpu.GetIdle())
	}

	fmt.Println(strings.Repeat("─", lineWidth))
}
