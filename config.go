package logger

import (
	"os"
	"path/filepath"
	"strings"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type LokiConfig struct {
	URL      string
	Username string
	Password string
	TenantID string
	Service  string
	Labels   map[string]string
	Enabled  *bool
}

type LoggerConfig struct {
	Loki     *LokiConfig
	LogLevel LogLevel
}

func resolveLogLevel(level LogLevel) logLevelResolved {
	if level != "" {
		return parseLogLevel(string(level))
	}

	fromEnv := os.Getenv("LOG_LEVEL")
	if fromEnv != "" {
		return parseLogLevel(fromEnv)
	}

	if strings.EqualFold(os.Getenv("ENV_ID"), "dev") {
		return logLevelResolved{raw: "debug", value: 5}
	}

	return logLevelResolved{raw: "info", value: 4}
}

func resolveLokiConfig(cfg LoggerConfig) *LokiConfig {
	if cfg.Loki == nil {
		if strings.TrimSpace(os.Getenv("LOKI_URL")) == "" {
			return nil
		}
		cfg.Loki = &LokiConfig{}
	}

	enabled := true
	if cfg.Loki.Enabled != nil {
		enabled = *cfg.Loki.Enabled
	}
	if !enabled {
		return nil
	}

	service := strings.TrimSpace(cfg.Loki.Service)
	if service == "" {
		service = strings.TrimSpace(os.Getenv("SERVICE_NAME"))
	}
	if service == "" {
		service = strings.TrimSpace(os.Getenv("COMPOSE_SERVICE"))
	}
	if service == "" {
		service = strings.TrimSpace(os.Getenv("npm_package_name"))
	}
	if service == "" {
		service = filepath.Base(mustGetwd())
	}

	url := strings.TrimSpace(cfg.Loki.URL)
	if url == "" {
		url = strings.TrimSpace(os.Getenv("LOKI_URL"))
	}
	if url == "" {
		url = "http://loki:3100/loki/api/v1/push"
	}

	labels := map[string]string{
		"service": service,
		"env":     envOrDefault("ENV_ID", "unknown"),
		"logger":  "log-lib-go",
	}
	for k, v := range cfg.Loki.Labels {
		labels[k] = v
	}

	return &LokiConfig{
		URL:      url,
		Username: cfg.Loki.Username,
		Password: cfg.Loki.Password,
		TenantID: cfg.Loki.TenantID,
		Service:  service,
		Labels:   labels,
		Enabled:  &enabled,
	}
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "service"
	}
	return wd
}
