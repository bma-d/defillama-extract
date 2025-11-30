package config

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoad_ValidAllFields(t *testing.T) {
	path := filepath.Join("testdata", "config_all.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Oracle.Name != "CustomOracle" {
		t.Errorf("Oracle.Name = %q, want %q", cfg.Oracle.Name, "CustomOracle")
	}
	if cfg.API.Timeout != 10*time.Second {
		t.Errorf("API.Timeout = %s, want %s", cfg.API.Timeout, 10*time.Second)
	}
	if cfg.API.MaxRetries != 5 {
		t.Errorf("API.MaxRetries = %d, want %d", cfg.API.MaxRetries, 5)
	}
	if cfg.API.RetryDelay != 2*time.Second {
		t.Errorf("API.RetryDelay = %s, want %s", cfg.API.RetryDelay, 2*time.Second)
	}
	if cfg.Output.Directory != "/tmp/data" {
		t.Errorf("Output.Directory = %q, want %q", cfg.Output.Directory, "/tmp/data")
	}
	if cfg.Scheduler.Interval != 30*time.Minute {
		t.Errorf("Scheduler.Interval = %s, want %s", cfg.Scheduler.Interval, 30*time.Minute)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, "debug")
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("Logging.Format = %q, want %q", cfg.Logging.Format, "text")
	}
}

func TestLoad_MinimalDefaultsApplied(t *testing.T) {
	path := filepath.Join("testdata", "config_minimal.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Oracle.Name != "Switchboard" {
		t.Errorf("Oracle.Name default = %q, want %q", cfg.Oracle.Name, "Switchboard")
	}
	if cfg.API.Timeout != 30*time.Second {
		t.Errorf("API.Timeout default = %s, want %s", cfg.API.Timeout, 30*time.Second)
	}
	if cfg.Output.Directory != "data" {
		t.Errorf("Output.Directory default = %q, want %q", cfg.Output.Directory, "data")
	}
	if cfg.Scheduler.StartImmediately != true {
		t.Errorf("Scheduler.StartImmediately default = %v, want true", cfg.Scheduler.StartImmediately)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "nope.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if got := err.Error(); !containsAny(got, []string{"config file not found", "no such file"}) {
		t.Fatalf("unexpected error message: %q", got)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "config_invalid.yaml"))
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	if !containsAny(err.Error(), []string{"failed to parse config", "yaml"}) {
		t.Fatalf("unexpected error message: %q", err)
	}
}

func TestValidate_InvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantMsg string
	}{
		{
			name:    "empty oracle name",
			mutate:  func(c *Config) { c.Oracle.Name = " " },
			wantMsg: "oracle.name",
		},
		{
			name:    "negative timeout",
			mutate:  func(c *Config) { c.API.Timeout = -1 },
			wantMsg: "api.timeout",
		},
		{
			name:    "negative max retries",
			mutate:  func(c *Config) { c.API.MaxRetries = -2 },
			wantMsg: "api.max_retries",
		},
		{
			name:    "negative retry delay",
			mutate:  func(c *Config) { c.API.RetryDelay = -1 },
			wantMsg: "api.retry_delay",
		},
		{
			name:    "invalid log level",
			mutate:  func(c *Config) { c.Logging.Level = "verbose" },
			wantMsg: "logging.level",
		},
		{
			name:    "invalid log format",
			mutate:  func(c *Config) { c.Logging.Format = "xml" },
			wantMsg: "logging.format",
		},
		{
			name:    "negative scheduler interval",
			mutate:  func(c *Config) { c.Scheduler.Interval = 0 },
			wantMsg: "scheduler.interval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			if err == nil || !containsAny(err.Error(), []string{tt.wantMsg}) {
				t.Fatalf("Validate() error = %v, want to contain %q", err, tt.wantMsg)
			}
		})
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := defaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate returned error for valid config: %v", err)
	}
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
