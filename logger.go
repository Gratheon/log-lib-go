package logger

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"
)

type logLevelResolved struct {
	raw   string
	value uint32
}

var (
	mu     sync.RWMutex
	global = buildLogger(LoggerConfig{})
)

func New(config LoggerConfig) *logrus.Logger {
	return buildLogger(config)
}

func Configure(config LoggerConfig) *logrus.Logger {
	configureTracing(config)
	instance := buildLogger(config)
	mu.Lock()
	global = instance
	mu.Unlock()
	return instance
}

// InitLogging keeps backward compatibility with existing swarm-api usage.
func InitLogging() *logrus.Logger {
	return Configure(LoggerConfig{})
}

func Info(message string, meta ...interface{}) {
	if len(meta) > 0 {
		active().WithFields(toFields(meta...)).Info(message)
		return
	}
	active().Info(message)
}

func Warn(message string, meta ...interface{}) {
	if len(meta) > 0 {
		active().WithFields(toFields(meta...)).Warn(message)
		return
	}
	active().Warn(message)
}

func Debug(message string, meta ...interface{}) {
	if len(meta) > 0 {
		active().WithFields(toFields(meta...)).Debug(message)
		return
	}
	active().Debug(message)
}

func Error(message string, meta ...interface{}) {
	if len(meta) > 0 {
		active().WithFields(toFields(meta...)).Error(message)
		return
	}
	active().Error(message)
}

func Fatal(message string, meta ...interface{}) {
	if len(meta) > 0 {
		active().WithFields(toFields(meta...)).Fatal(message)
		return
	}
	active().Fatal(message)
}

func ErrorEnriched(message string, err error, meta map[string]interface{}) {
	fields := logrus.Fields{}
	for k, v := range meta {
		fields[k] = v
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	active().WithFields(fields).Error(message)
}

func ErrorWithRequest(r *http.Request, message string) {
	fields := logrus.Fields{}
	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		fields["req_id"] = reqID
	}
	if userID, ok := r.Context().Value("userID").(string); ok {
		fields["user_id"] = userID
	}
	addTraceContextFields(fields, r.Context())
	active().WithFields(fields).Error(message)
}

func ErrorWithContext(ctx context.Context, message string) {
	fields := logrus.Fields{}
	if reqID := middleware.GetReqID(ctx); reqID != "" {
		fields["req_id"] = reqID
	}
	if userID, ok := ctx.Value("userID").(string); ok {
		fields["user_id"] = userID
	}
	addTraceContextFields(fields, ctx)
	active().WithFields(fields).Error(message)
}

func toFields(meta ...interface{}) logrus.Fields {
	fields := logrus.Fields{}
	for i, m := range meta {
		fields["meta_"+strconv.Itoa(i)] = m
	}
	return fields
}

func buildLogger(config LoggerConfig) *logrus.Logger {
	level := resolveLogLevel(config.LogLevel)
	instance := logrus.New()
	instance.SetLevel(level.toLogrus())
	instance.SetFormatter(&ColoredFormatter{})

	if otlpCfg := resolveOtlpConfig(config); otlpCfg != nil {
		instance.AddHook(newOtlpTarget(*otlpCfg))
	}

	return instance
}

func active() *logrus.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return global
}

func parseLogLevel(raw string) logLevelResolved {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug", "trace":
		return logLevelResolved{raw: "debug", value: 5}
	case "warn", "warning":
		return logLevelResolved{raw: "warn", value: 3}
	case "error", "fatal", "panic":
		return logLevelResolved{raw: "error", value: 2}
	default:
		return logLevelResolved{raw: "info", value: 4}
	}
}

func (l logLevelResolved) toLogrus() logrus.Level {
	switch l.raw {
	case "debug":
		return logrus.DebugLevel
	case "warn":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	default:
		return logrus.InfoLevel
	}
}
