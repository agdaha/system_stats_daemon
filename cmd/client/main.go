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

const (
	subsysLoadAvg    = "loadavg"
	subsysCPU        = "cpu"
	subsysDiskIO     = "diskio"
	subsysFilesystem = "filesystem"
	subsysNetTraffic = "nettraffic"
	subsysNetSockets = "netsockets"

	lineWidth = 60
)

func main() {
	addr := flag.String("addr", "localhost:50051", "daemon address host:port")
	n := flag.Uint("n", 5, "snapshot interval in seconds")
	m := flag.Uint("m", 15, "averaging window in seconds")
	subsystem := flag.String("subsystem", "",
		"comma-separated subsystems to display (default: all).\n"+
			"  values: loadavg, cpu, diskio, filesystem, nettraffic, netsockets")
	flag.Parse()

	if *n == 0 || *m == 0 {
		log.Fatal("--n and --m must be greater than 0")
	}

	if err := run(*addr, uint32(*n), uint32(*m), parseFilter(*subsystem)); err != nil { //nolint:gosec // validated
		log.Fatal(err)
	}
}

// parseFilter converts a comma-separated subsystem list into a lookup set.
// Returns nil when s is empty, which means "show all subsystems".
func parseFilter(s string) map[string]bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	f := make(map[string]bool)
	for _, part := range strings.Split(s, ",") {
		if name := strings.TrimSpace(part); name != "" {
			f[name] = true
		}
	}
	return f
}

func run(addr string, n, m uint32, filter map[string]bool) error {
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

	p := &printer{filter: filter}
	for {
		snap, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("recv: %w", err)
		}
		p.printSnapshot(snap, addr, n, m)
	}
}

type printer struct {
	filter map[string]bool // nil means show all subsystems
}

// show reports whether the named subsystem should be displayed.
func (p *printer) show(subsys string) bool {
	return p.filter == nil || p.filter[subsys]
}

func (p *printer) printSnapshot(snap *statspb.Snapshot, addr string, n, m uint32) {
	fmt.Print("\033[2J\033[H") // clear screen, move cursor to top
	fmt.Printf("sysmon  %s  N=%ds M=%ds  %s\n",
		addr, n, m, time.Now().Format("15:04:05"))
	fmt.Println(strings.Repeat("─", lineWidth))
	p.printSystem(snap)
	p.printDisk(snap)
	p.printNetwork(snap)
	fmt.Println(strings.Repeat("─", lineWidth))
}

func (p *printer) printSystem(snap *statspb.Snapshot) {
	if p.show(subsysLoadAvg) {
		if la := snap.GetLoadAverage(); la != nil {
			fmt.Println("LOAD AVERAGE")
			fmt.Printf("   1 min  %8.2f\n", la.GetOne())
			fmt.Printf("   5 min  %8.2f\n", la.GetFive())
			fmt.Printf("  15 min  %8.2f\n", la.GetFifteen())
		}
	}
	if p.show(subsysCPU) {
		if cpu := snap.GetCpu(); cpu != nil {
			fmt.Println("CPU")
			fmt.Printf("   user   %7.1f%%\n", cpu.GetUserMode())
			fmt.Printf("   system %7.1f%%\n", cpu.GetSystemMode())
			fmt.Printf("   idle   %7.1f%%\n", cpu.GetIdle())
		}
	}
}

func (p *printer) printDisk(snap *statspb.Snapshot) {
	if p.show(subsysDiskIO) {
		if disks := snap.GetDiskIo(); len(disks) > 0 {
			fmt.Println("DISK I/O")
			fmt.Printf("   %-12s %8s %10s\n", "device", "tps", "KB/s")
			for _, d := range disks {
				fmt.Printf("   %-12s %8.1f %10.1f\n", d.GetDevice(), d.GetTps(), d.GetKbps())
			}
		}
	}
	if p.show(subsysFilesystem) {
		if fss := snap.GetFilesystems(); len(fss) > 0 {
			fmt.Println("FILESYSTEMS")
			fmt.Printf("   %-20s %8s %6s %10s %6s\n", "mount", "used MB", "use%", "inodes", "ino%")
			for _, fs := range fss {
				fmt.Printf("   %-20s %8d %5.0f%% %10d %5.0f%%\n",
					fs.GetMountPoint(), fs.GetUsedMb(), fs.GetUsedPercent(),
					fs.GetUsedInodes(), fs.GetUsedInodesPercent())
			}
		}
	}
}

func (p *printer) printNetwork(snap *statspb.Snapshot) {
	net := snap.GetNetwork()
	if net == nil {
		return
	}
	if p.show(subsysNetTraffic) {
		p.printNetInterfaces(net.GetProtocolTalkers())
		p.printNetFlows(net.GetTrafficTalkers())
	}
	if p.show(subsysNetSockets) {
		p.printNetSockets(net.GetListeningSockets())
		p.printTCPStates(net.GetTcpStates())
	}
}

func (p *printer) printNetInterfaces(protos []*statspb.ProtocolTalker) {
	if len(protos) == 0 {
		return
	}
	fmt.Println("NETWORK INTERFACES")
	fmt.Printf("   %-16s %10s %6s\n", "interface", "bytes/s", "pct")
	for _, pt := range protos {
		fmt.Printf("   %-16s %10d %5.1f%%\n", pt.GetProtocol(), pt.GetBytes(), pt.GetPercent())
	}
}

func (p *printer) printNetFlows(flows []*statspb.TrafficTalker) {
	if len(flows) == 0 {
		return
	}
	fmt.Println("TOP FLOWS")
	fmt.Printf("   %-15s %-15s %-6s %8s\n", "src", "dst", "proto", "BPS")
	for i, f := range flows {
		if i >= 10 {
			break
		}
		fmt.Printf("   %-15s %-15s %-6s %8.0f\n",
			f.GetSrcAddr(), f.GetDstAddr(), f.GetProtocol(), f.GetBps())
	}
}

func (p *printer) printNetSockets(socks []*statspb.ListeningSocket) {
	if len(socks) == 0 {
		return
	}
	fmt.Println("LISTENING SOCKETS")
	fmt.Printf("   %-6s %6s  %-20s %8s\n", "proto", "port", "command", "pid")
	for _, s := range socks {
		fmt.Printf("   %-6s %6d  %-20s %8d\n",
			s.GetProtocol(), s.GetPort(), s.GetCommand(), s.GetPid())
	}
}

func (p *printer) printTCPStates(states []*statspb.TCPState) {
	if len(states) == 0 {
		return
	}
	fmt.Println("TCP STATES")
	for _, st := range states {
		fmt.Printf("   %-12s %d\n", st.GetState(), st.GetCount())
	}
}
