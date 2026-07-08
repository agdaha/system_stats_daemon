package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type Subsystems struct {
	LoadAverage bool `json:"load_average"`
	CPU         bool `json:"cpu"`
	DiskIO      bool `json:"disk_io"`
	Filesystem  bool `json:"filesystem"`
	NetTraffic  bool `json:"net_traffic"`
	NetSockets  bool `json:"net_sockets"`
}

type Config struct {
	Port       int
	ConfigFile string
	Subsystems Subsystems
}

func Load(args []string) (*Config, error) {
	fs := flag.NewFlagSet("daemon", flag.ContinueOnError)
	port := fs.Int("port", 50051, "gRPC server port")
	cfgFile := fs.String("config", "", "path to JSON config file")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parse flags: %w", err)
	}

	cfg := &Config{
		Port:       *port,
		ConfigFile: *cfgFile,
		Subsystems: defaultSubsystems(),
	}

	if *cfgFile != "" {
		if err := loadFile(*cfgFile, cfg); err != nil {
			return nil, fmt.Errorf("load config file %q: %w", *cfgFile, err)
		}
	}

	return cfg, nil
}

func defaultSubsystems() Subsystems {
	return Subsystems{
		LoadAverage: true,
		CPU:         true,
		DiskIO:      true,
		Filesystem:  true,
		NetTraffic:  true,
		NetSockets:  true,
	}
}

func loadFile(path string, cfg *Config) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var fileCfg struct {
		Subsystems Subsystems `json:"subsystems"`
	}
	if err := json.NewDecoder(f).Decode(&fileCfg); err != nil {
		return err
	}

	cfg.Subsystems = fileCfg.Subsystems
	return nil
}
