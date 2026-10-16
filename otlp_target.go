package logger

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
)

type otlpTarget struct {
	logger otellog.Logger
}

func newOtlpTarget(config OtlpConfig) *otlpTarget {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exporter, err := otlploghttp.New(
		ctx,
		otlploghttp.WithEndpointURL(config.Endpoint),
		otlploghttp.WithHeaders(config.Headers),
	)
	if err != nil {
		return &otlpTarget{logger: nil}
	}

	attrs := make([]attribute.KeyValue, 0, len(config.ResourceAttributes))
	for k, v := range config.ResourceAttributes {
		attrs = append(attrs, attribute.String(k, v))
	}
	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(resource.NewWithAttributes("", attrs...)),
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(exporter)),
	)

	return &otlpTarget{logger: provider.Logger("gratheon-log-lib-go", otellog.WithInstrumentationVersion(LibraryVersion))}
}

func (o *otlpTarget) Levels() []logrus.Level {
	return []logrus.Level{logrus.InfoLevel, logrus.WarnLevel, logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel}
}

func (o *otlpTarget) Fire(entry *logrus.Entry) error {
	if o.logger == nil {
		return nil
	}

	attrs := make([]attribute.KeyValue, 0, len(entry.Data)+1)
	for k, v := range entry.Data {
		if isTraceField(k) {
			continue
		}
		attrs = append(attrs, otlpAttribute(k, v))
	}
	attrs = append(attrs, attribute.String("log.logger", "log-lib-go"))

	var record otellog.Record
	record.SetTimestamp(entry.Time)
	record.SetObservedTimestamp(time.Now())
	record.SetSeverity(severity(entry.Level))
	record.SetSeverityText(strings.ToUpper(entry.Level.String()))
	record.SetBody(attribute.StringValue(entry.Message))
	record.AddAttributes(attrs...)

	o.logger.Emit(contextFromTraceFields(entry.Data), record)
	return nil
}

func contextFromTraceFields(fields logrus.Fields) context.Context {
	traceID, _ := fields["trace_id"].(string)
	if traceID == "" {
		traceID, _ = fields["traceId"].(string)
	}
	spanID, _ := fields["span_id"].(string)
	if spanID == "" {
		spanID, _ = fields["spanId"].(string)
	}

	parsedTraceID, traceErr := trace.TraceIDFromHex(traceID)
	parsedSpanID, spanErr := trace.SpanIDFromHex(spanID)
	if traceErr != nil || spanErr != nil {
		return context.Background()
	}

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: parsedTraceID,
		SpanID:  parsedSpanID,
		Remote:  true,
	})
	return trace.ContextWithSpanContext(context.Background(), spanContext)
}

func severity(level logrus.Level) otellog.Severity {
	switch level {
	case logrus.DebugLevel, logrus.TraceLevel:
		return otellog.SeverityDebug
	case logrus.WarnLevel:
		return otellog.SeverityWarn
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return otellog.SeverityError
	default:
		return otellog.SeverityInfo
	}
}

func otlpAttribute(key string, value interface{}) attribute.KeyValue {
	switch v := value.(type) {
	case nil:
		return attribute.String(key, "")
	case string:
		return attribute.String(key, v)
	case bool:
		return attribute.Bool(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case float32:
		return attribute.Float64(key, float64(v))
	default:
		return attribute.String(key, fmt.Sprintf("%v", v))
	}
}

func isTraceField(key string) bool {
	return key == "trace_id" || key == "traceId" || key == "span_id" || key == "spanId"
}
