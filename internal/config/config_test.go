package config_test

import (
	"os"
	"testing"

	"system_stats_deamon/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := config.Load([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 50051 {
		t.Errorf("want port 50051, got %d", cfg.Port)
	}
	sub := cfg.Subsystems
	if !sub.LoadAverage || !sub.CPU || !sub.DiskIO || !sub.Filesystem || !sub.NetTraffic || !sub.NetSockets {
		t.Error("all subsystems should be enabled by default")
	}
}

func TestLoad_PortFlag(t *testing.T) {
	cfg, err := config.Load([]string{"--port", "8080"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("want port 8080, got %d", cfg.Port)
	}
}

func TestLoad_ConfigFile(t *testing.T) {
	const content = `{"subsystems":{"load_average":false,"cpu":true}}`

	f, err := os.CreateTemp(t.TempDir(), "cfg*.json")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err = f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	f.Close()

	cfg, err := config.Load([]string{"--config", f.Name()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Subsystems.LoadAverage {
		t.Error("LoadAverage should be disabled")
	}
	if !cfg.Subsystems.CPU {
		t.Error("CPU should be enabled")
	}
}

func TestLoad_MissingConfigFile(t *testing.T) {
	_, err := config.Load([]string{"--config", "/xxx/config.json"})
	if err == nil {
		t.Error("expected error for missing config file")
	}
}
