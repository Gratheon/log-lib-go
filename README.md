# log-lib-go

Reusable Go logging library for Gratheon services with colored console output and optional OTLP log export to ClickStack/OpenTelemetry collectors.

## Version

`LibraryVersion` is `2.0.0`, the ClickStack/OTLP major release. Loki support has been removed.

## Features

- Colored console logger for `info`, `warn`, `error`, and `debug`
- Optional OTLP HTTP JSON log export
- Request/context correlation helpers (`req_id`, `user_id`)
- `chi` request logging middleware support
- Backward-compatible package-level API (`Info`, `Error`, etc.)

## ClickStack Setup

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_SERVICE_NAME=swarm-api
OTEL_RESOURCE_ATTRIBUTES=deployment.environment.name=dev,service.namespace=gratheon
```

The library sends logs to `/v1/logs`, so either a collector base endpoint or a full logs endpoint works.

## Usage

```go
package main

import logger "github.com/Gratheon/log-lib-go"

func main() {
	logger.Configure(logger.LoggerConfig{
		LogLevel: logger.LogLevelInfo,
		Otlp: &logger.OtlpConfig{
			Endpoint: "http://localhost:4318",
			Service:  "swarm-api",
			ResourceAttributes: map[string]string{
				"deployment.environment.name": "dev",
			},
		},
	})

	logger.Info("service started")
	logger.Warn("slow request", map[string]interface{}{"route": "/graphql", "duration_ms": 830})
}
```

Trace correlation is supported when log fields include `trace_id`/`traceId` and `span_id`/`spanId`.
