package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	statspb "system_stats_deamon/api/stats"
	"system_stats_deamon/internal/config"
	"system_stats_deamon/internal/server"
)

func main() {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	srv := grpc.NewServer()
	statspb.RegisterStatsServiceServer(srv, server.New())

	log.Printf("daemon listening on :%d", cfg.Port)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
