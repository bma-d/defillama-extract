# 10. Configuration & Environment

## 10.1 Configuration File (YAML)

```yaml
# config.yaml

# Oracle settings
oracle:
  name: "Switchboard"
  website: "https://switchboard.xyz"
  documentation: "https://docs.switchboard.xyz"

# API settings
api:
  oracle_url: "https://api.llama.fi/oracles"
  protocols_url: "https://api.llama.fi/lite/protocols2?b=2"
  timeout: 30s
  max_retries: 3
  retry_base_delay: 1s
  retry_max_delay: 30s
  user_agent: "SwitchboardOracleExtractor/1.0"

# Output settings
output:
  directory: "./data"
  full_file: "switchboard-oracle-data.json"
  min_file: "switchboard-oracle-data.min.json"
  summary_file: "switchboard-summary.json"
  state_file: "state.json"

# History settings
history:
  retention_days: 90
  max_snapshots: 2160

# Scheduler settings (for daemon mode)
scheduler:
  enabled: true
  interval: 15m
  start_immediately: true

# Logging settings
logging:
  level: "info"
  format: "json"
  output: "stdout"

# Monitoring settings
monitoring:
  enabled: true
  port: 9090
  path: "/metrics"
```

## 10.2 Configuration Loading

```go
// internal/config/config.go

package config

import (
    "fmt"
    "os"
    "time"

    "gopkg.in/yaml.v3"
)

type Config struct {
    Oracle     OracleConfig     `yaml:"oracle"`
    API        APIConfig        `yaml:"api"`
    Output     OutputConfig     `yaml:"output"`
    History    HistoryConfig    `yaml:"history"`
    Scheduler  SchedulerConfig  `yaml:"scheduler"`
    Logging    LoggingConfig    `yaml:"logging"`
    Monitoring MonitoringConfig `yaml:"monitoring"`
}

type OracleConfig struct {
    Name          string `yaml:"name"`
    Website       string `yaml:"website"`
    Documentation string `yaml:"documentation"`
}

type APIConfig struct {
    OracleURL      string        `yaml:"oracle_url"`
    ProtocolsURL   string        `yaml:"protocols_url"`
    Timeout        time.Duration `yaml:"timeout"`
    MaxRetries     int           `yaml:"max_retries"`
    RetryBaseDelay time.Duration `yaml:"retry_base_delay"`
    RetryMaxDelay  time.Duration `yaml:"retry_max_delay"`
    UserAgent      string        `yaml:"user_agent"`
}

type OutputConfig struct {
    Directory   string `yaml:"directory"`
    FullFile    string `yaml:"full_file"`
    MinFile     string `yaml:"min_file"`
    SummaryFile string `yaml:"summary_file"`
    StateFile   string `yaml:"state_file"`
}

type HistoryConfig struct {
    RetentionDays int `yaml:"retention_days"`
    MaxSnapshots  int `yaml:"max_snapshots"`
}

type SchedulerConfig struct {
    Enabled          bool          `yaml:"enabled"`
    Interval         time.Duration `yaml:"interval"`
    StartImmediately bool          `yaml:"start_immediately"`
}

type LoggingConfig struct {
    Level  string `yaml:"level"`
    Format string `yaml:"format"`
    Output string `yaml:"output"`
}

type MonitoringConfig struct {
    Enabled bool   `yaml:"enabled"`
    Port    int    `yaml:"port"`
    Path    string `yaml:"path"`
}

// Load reads configuration from file and applies environment overrides
func Load(path string) (*Config, error) {
    cfg := DefaultConfig()

    // Load from file if exists
    if path != "" {
        data, err := os.ReadFile(path)
        if err != nil && !os.IsNotExist(err) {
            return nil, fmt.Errorf("reading config file: %w", err)
        }
        if err == nil {
            if err := yaml.Unmarshal(data, cfg); err != nil {
                return nil, fmt.Errorf("parsing config file: %w", err)
            }
        }
    }

    // Apply environment variable overrides
    applyEnvOverrides(cfg)

    return cfg, nil
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
    return &Config{
        Oracle: OracleConfig{
            Name:    "Switchboard",
            Website: "https://switchboard.xyz",
        },
        API: APIConfig{
            OracleURL:      "https://api.llama.fi/oracles",
            ProtocolsURL:   "https://api.llama.fi/lite/protocols2?b=2",
            Timeout:        30 * time.Second,
            MaxRetries:     3,
            RetryBaseDelay: 1 * time.Second,
            RetryMaxDelay:  30 * time.Second,
            UserAgent:      "SwitchboardOracleExtractor/1.0",
        },
        Output: OutputConfig{
            Directory:   "./data",
            FullFile:    "switchboard-oracle-data.json",
            MinFile:     "switchboard-oracle-data.min.json",
            SummaryFile: "switchboard-summary.json",
            StateFile:   "state.json",
        },
        History: HistoryConfig{
            RetentionDays: 90,
            MaxSnapshots:  2160,
        },
        Scheduler: SchedulerConfig{
            Enabled:          true,
            Interval:         15 * time.Minute,
            StartImmediately: true,
        },
        Logging: LoggingConfig{
            Level:  "info",
            Format: "json",
            Output: "stdout",
        },
        Monitoring: MonitoringConfig{
            Enabled: true,
            Port:    9090,
            Path:    "/metrics",
        },
    }
}

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
    // Add more overrides as needed
}
```

---
