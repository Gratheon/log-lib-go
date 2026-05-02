package logger

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func configureTracing(config LoggerConfig) {
	traceConfig := resolveTraceConfig(config)
	if traceConfig == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpointURL(traceConfig.Endpoint),
		otlptracehttp.WithHeaders(traceConfig.Headers),
	)
	if err != nil {
		return
	}

	attrs := make([]attribute.KeyValue, 0, len(traceConfig.ResourceAttributes))
	for k, v := range traceConfig.ResourceAttributes {
		attrs = append(attrs, attribute.String(k, v))
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(sdkresource.NewWithAttributes("", attrs...)),
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
}

func resolveTraceConfig(cfg LoggerConfig) *OtlpConfig {
	explicit := cfg.Traces
	if explicit == nil {
		explicit = cfg.Otlp
	}

	endpoint := ""
	if explicit != nil {
		endpoint = strings.TrimSpace(explicit.Endpoint)
	}
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"))
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
		service = "service"
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
	for k, v := range parseKeyValueList(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS")) {
		headers[k] = v
	}
	if explicit != nil {
		for k, v := range explicit.Headers {
			headers[k] = v
		}
	}

	return &OtlpConfig{
		Endpoint:           normalizeOtlpTracesEndpoint(endpoint),
		Headers:            headers,
		Service:            service,
		ResourceAttributes: attrs,
		Enabled:            &enabled,
	}
}

func NewTraceMiddleware(operation string) func(http.Handler) http.Handler {
	configureTracing(LoggerConfig{})
	if strings.TrimSpace(operation) == "" {
		operation = envOrDefault("OTEL_SERVICE_NAME", "http.server")
	}
	return otelhttp.NewMiddleware(operation)
}

func TraceContextFields(ctx context.Context) logrus.Fields {
	fields := logrus.Fields{}
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return fields
	}
	fields["trace_id"] = spanContext.TraceID().String()
	fields["span_id"] = spanContext.SpanID().String()
	return fields
}

func addTraceContextFields(fields logrus.Fields, ctx context.Context) {
	for key, value := range TraceContextFields(ctx) {
		fields[key] = value
	}
}
