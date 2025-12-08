package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration sections loaded from YAML.
type Config struct {
	Oracle    OracleConfig    `yaml:"oracle"`
	API       APIConfig       `yaml:"api"`
	Output    OutputConfig    `yaml:"output"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Logging   LoggingConfig   `yaml:"logging"`
	TVL       TVLConfig       `yaml:"tvl"`
}

type OracleConfig struct {
	Name          string `yaml:"name"`
	Website       string `yaml:"website"`
	Documentation string `yaml:"documentation"`
}

type APIConfig struct {
	OraclesURL   string        `yaml:"oracles_url"`
	ProtocolsURL string        `yaml:"protocols_url"`
	Timeout      time.Duration `yaml:"timeout"`
	MaxRetries   int           `yaml:"max_retries"`
	RetryDelay   time.Duration `yaml:"retry_delay"`
}

type OutputConfig struct {
	Directory   string `yaml:"directory"`
	FullFile    string `yaml:"full_file"`
	MinFile     string `yaml:"min_file"`
	SummaryFile string `yaml:"summary_file"`
	StateFile   string `yaml:"state_file"`
}

type SchedulerConfig struct {
	Interval         time.Duration `yaml:"interval"`
	StartImmediately bool          `yaml:"start_immediately"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type TVLConfig struct {
	CustomProtocolsPath string `yaml:"custom_protocols_path"`
	Enabled             bool   `yaml:"enabled"`
}

// applyEnvOverrides applies environment variable overrides to the provided config in place.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("ORACLE_NAME"); v != "" {
		cfg.Oracle.Name = v
	}

	if v := os.Getenv("OUTPUT_DIR"); v != "" {
		cfg.Output.Directory = v
	}

	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}

	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.Logging.Format = v
	}

	if v := os.Getenv("API_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.API.Timeout = d
		} else {
			log.Printf("warning: invalid API_TIMEOUT %q, using YAML/default value", v)
		}
	}

	if v := os.Getenv("SCHEDULER_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Scheduler.Interval = d
		} else {
			log.Printf("warning: invalid SCHEDULER_INTERVAL %q, using YAML/default value", v)
		}
	}

	if v := os.Getenv("TVL_CUSTOM_PROTOCOLS_PATH"); v != "" {
		cfg.TVL.CustomProtocolsPath = v
	}
	if v := os.Getenv("TVL_ENABLED"); v != "" {
		cfg.TVL.Enabled = strings.ToLower(v) == "true"
	}
}

// defaultConfig returns configuration populated with documented defaults.
func defaultConfig() Config {
	return Config{
		Oracle: OracleConfig{
			Name:          "Switchboard",
			Website:       "",
			Documentation: "",
		},
		API: APIConfig{
			OraclesURL:   "https://api.llama.fi/oracles",
			ProtocolsURL: "https://api.llama.fi/lite/protocols2?b=2",
			Timeout:      30 * time.Second,
			MaxRetries:   3,
			RetryDelay:   1 * time.Second,
		},
		Output: OutputConfig{
			Directory:   "data",
			FullFile:    "switchboard-oracle-data.json",
			MinFile:     "switchboard-oracle-data.min.json",
			SummaryFile: "switchboard-summary.json",
			StateFile:   "state.json",
		},
		Scheduler: SchedulerConfig{
			Interval:         2 * time.Hour,
			StartImmediately: true,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		TVL: TVLConfig{
			CustomProtocolsPath: "config/custom-protocols.json",
			Enabled:             true,
		},
	}
}

// Load reads YAML configuration from path, applying defaults and returning a populated Config.
func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config file not found: %w", err)
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	applyEnvOverrides(&cfg)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks configuration values and returns the first encountered error.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Oracle.Name) == "" {
		return errors.New("oracle.name must not be empty")
	}
	if c.API.Timeout <= 0 {
		return fmt.Errorf("api.timeout must be positive, got %s", c.API.Timeout)
	}
	if c.API.MaxRetries < 0 {
		return fmt.Errorf("api.max_retries must be non-negative, got %d", c.API.MaxRetries)
	}
	if c.API.RetryDelay < 0 {
		return fmt.Errorf("api.retry_delay must be non-negative, got %s", c.API.RetryDelay)
	}
	if c.Scheduler.Interval <= 0 {
		return fmt.Errorf("scheduler.interval must be positive, got %s", c.Scheduler.Interval)
	}

	validLevels := map[string]struct{}{
		"debug": {},
		"info":  {},
		"warn":  {},
		"error": {},
	}
	if _, ok := validLevels[strings.ToLower(c.Logging.Level)]; !ok {
		return fmt.Errorf("logging.level must be one of debug, info, warn, error; got %q", c.Logging.Level)
	}

	validFormats := map[string]struct{}{
		"json": {},
		"text": {},
	}
	if _, ok := validFormats[strings.ToLower(c.Logging.Format)]; !ok {
		return fmt.Errorf("logging.format must be one of json, text; got %q", c.Logging.Format)
	}

	if strings.TrimSpace(c.TVL.CustomProtocolsPath) == "" {
		return errors.New("tvl.custom_protocols_path must not be empty")
	}

	return nil
}
