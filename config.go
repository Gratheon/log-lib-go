package logger

import (
	"os"
	"path/filepath"
	"strings"
)

const LibraryVersion = "2.1.0"

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type OtlpConfig struct {
	Endpoint           string
	Headers            map[string]string
	Service            string
	ResourceAttributes map[string]string
	Enabled            *bool
}

type LoggerConfig struct {
	Otlp     *OtlpConfig
	Traces   *OtlpConfig
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

func resolveOtlpConfig(cfg LoggerConfig) *OtlpConfig {
	explicit := cfg.Otlp
	endpoint := ""
	if explicit != nil {
		endpoint = strings.TrimSpace(explicit.Endpoint)
	}
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT"))
	}
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	}

	enabled := endpoint != ""
	if explicit != nil && explicit.Enabled != nil {
		enabled = *explicit.Enabled
	}
	if !enabled || endpoint == "" {
		return nil
	}

	service := ""
	if explicit != nil {
		service = strings.TrimSpace(explicit.Service)
	}
	if service == "" {
		service = strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME"))
	}
	if service == "" {
		service = strings.TrimSpace(os.Getenv("SERVICE_NAME"))
	}
	if service == "" {
		service = strings.TrimSpace(os.Getenv("COMPOSE_SERVICE"))
	}
	if service == "" {
		service = filepath.Base(mustGetwd())
	}

	attrs := map[string]string{
		"service.name":                service,
		"service.namespace":           "gratheon",
		"deployment.environment.name": envOrDefault("ENV_ID", "unknown"),
	}
	for k, v := range parseKeyValueList(os.Getenv("OTEL_RESOURCE_ATTRIBUTES")) {
		attrs[k] = v
	}
	if explicit != nil {
		for k, v := range explicit.ResourceAttributes {
			attrs[k] = v
		}
	}

	headers := map[string]string{}
	if apiKey := strings.TrimSpace(os.Getenv("HYPERDX_API_KEY")); apiKey != "" {
		headers["authorization"] = apiKey
	}
	for k, v := range parseKeyValueList(os.Getenv("OTEL_EXPORTER_OTLP_HEADERS")) {
		headers[k] = v
	}
	for k, v := range parseKeyValueList(os.Getenv("OTEL_EXPORTER_OTLP_LOGS_HEADERS")) {
		headers[k] = v
	}
	if explicit != nil {
		for k, v := range explicit.Headers {
			headers[k] = v
		}
	}

	return &OtlpConfig{
		Endpoint:           normalizeOtlpLogsEndpoint(endpoint),
		Headers:            headers,
		Service:            service,
		ResourceAttributes: attrs,
		Enabled:            &enabled,
	}
}

func normalizeOtlpLogsEndpoint(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if strings.HasSuffix(endpoint, "/v1/logs") {
		return endpoint
	}
	return endpoint + "/v1/logs"
}

func normalizeOtlpTracesEndpoint(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if strings.HasSuffix(endpoint, "/v1/traces") {
		return endpoint
	}
	return endpoint + "/v1/traces"
}

func parseKeyValueList(raw string) map[string]string {
	values := map[string]string{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			continue
		}
		values[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return values
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
