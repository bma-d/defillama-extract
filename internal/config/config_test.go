package config

import (
	"bytes"
	"log"
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

func TestLoad_EnvOverrides_StringAndDuration(t *testing.T) {
	path := filepath.Join("testdata", "config_minimal.yaml")

	t.Setenv("ORACLE_NAME", "EnvOracle")
	t.Setenv("OUTPUT_DIR", "/env/out")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")
	t.Setenv("API_TIMEOUT", "45s")
	t.Setenv("SCHEDULER_INTERVAL", "1h30m")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Oracle.Name != "EnvOracle" {
		t.Errorf("Oracle.Name = %q, want %q", cfg.Oracle.Name, "EnvOracle")
	}
	if cfg.Output.Directory != "/env/out" {
		t.Errorf("Output.Directory = %q, want %q", cfg.Output.Directory, "/env/out")
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, "debug")
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("Logging.Format = %q, want %q", cfg.Logging.Format, "text")
	}
	if cfg.API.Timeout != 45*time.Second {
		t.Errorf("API.Timeout = %s, want %s", cfg.API.Timeout, 45*time.Second)
	}
	if cfg.Scheduler.Interval != time.Hour+30*time.Minute {
		t.Errorf("Scheduler.Interval = %s, want %s", cfg.Scheduler.Interval, time.Hour+30*time.Minute)
	}
}

func TestLoad_EnvOverrides_PrecedenceOverYAML(t *testing.T) {
	path := filepath.Join("testdata", "config_all.yaml")

	t.Setenv("ORACLE_NAME", "EnvWins")
	t.Setenv("OUTPUT_DIR", "/env/dir")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("API_TIMEOUT", "99s")
	t.Setenv("SCHEDULER_INTERVAL", "15m")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Oracle.Name != "EnvWins" {
		t.Errorf("Oracle.Name = %q, want env override %q", cfg.Oracle.Name, "EnvWins")
	}
	if cfg.Output.Directory != "/env/dir" {
		t.Errorf("Output.Directory = %q, want env override %q", cfg.Output.Directory, "/env/dir")
	}
	if cfg.Logging.Level != "warn" {
		t.Errorf("Logging.Level = %q, want env override %q", cfg.Logging.Level, "warn")
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("Logging.Format = %q, want env override %q", cfg.Logging.Format, "json")
	}
	if cfg.API.Timeout != 99*time.Second {
		t.Errorf("API.Timeout = %s, want env override %s", cfg.API.Timeout, 99*time.Second)
	}
	if cfg.Scheduler.Interval != 15*time.Minute {
		t.Errorf("Scheduler.Interval = %s, want env override %s", cfg.Scheduler.Interval, 15*time.Minute)
	}
}

func TestLoad_InvalidDurationEnvVars_LogAndFallback(t *testing.T) {
	path := filepath.Join("testdata", "config_all.yaml")

	buf := &bytes.Buffer{}
	origWriter := log.Writer()
	log.SetOutput(buf)
	t.Cleanup(func() { log.SetOutput(origWriter) })

	t.Setenv("API_TIMEOUT", "notaduration")
	t.Setenv("SCHEDULER_INTERVAL", "bad")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.API.Timeout != 10*time.Second { // YAML value should remain
		t.Errorf("API.Timeout = %s, want YAML value %s", cfg.API.Timeout, 10*time.Second)
	}
	if cfg.Scheduler.Interval != 30*time.Minute {
		t.Errorf("Scheduler.Interval = %s, want YAML value %s", cfg.Scheduler.Interval, 30*time.Minute)
	}

	out := buf.String()
	if !strings.Contains(out, "invalid API_TIMEOUT") {
		t.Errorf("log output missing API_TIMEOUT warning: %q", out)
	}
	if !strings.Contains(out, "invalid SCHEDULER_INTERVAL") {
		t.Errorf("log output missing SCHEDULER_INTERVAL warning: %q", out)
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
