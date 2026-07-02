package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"google.golang.org/grpc"

	statspb "system_stats_deamon/api/stats"
	"system_stats_deamon/internal/collector/cpu"
	"system_stats_deamon/internal/collector/loadavg"
	"system_stats_deamon/internal/config"
	"system_stats_deamon/internal/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	deps := buildDeps(ctx, cfg)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	srv := grpc.NewServer()
	statspb.RegisterStatsServiceServer(srv, server.New(deps))

	go func() {
		<-ctx.Done()
		srv.GracefulStop()
	}()

	log.Printf("daemon listening on :%d", cfg.Port)
	return srv.Serve(lis)
}

func buildDeps(ctx context.Context, cfg *config.Config) server.Deps {
	deps := server.Deps{}

	if cfg.Subsystems.LoadAverage {
		la := loadavg.New()
		la.Start(ctx)
		deps.LoadAvg = la
	}

	if cfg.Subsystems.CPU {
		cp := cpu.New()
		cp.Start(ctx)
		deps.CPU = cp
	}

	return deps
}
