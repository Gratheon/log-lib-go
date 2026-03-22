# log-lib-go

Reusable Go logging library for Gratheon services.

## Features

- Colored console logger (info/warn/error/debug)
- Optional Loki target (`LOKI_URL` or explicit config)
- Request/context correlation helpers (`req_id`, `user_id`)
- `chi` request logging middleware support
- Backward-compatible package-level API (`Info`, `Error`, etc.)

## Install

```bash
go get github.com/Gratheon/log-lib-go
```

## Usage

```go
package main

import logger "github.com/Gratheon/log-lib-go"

func main() {
	logger.Configure(logger.LoggerConfig{
		LogLevel: logger.LogLevelInfo,
		Loki: &logger.LokiConfig{
			Enabled: boolPtr(true),
			URL:     "http://loki:3100/loki/api/v1/push",
			Service: "swarm-api",
			Labels: map[string]string{
				"team": "platform",
			},
		},
	})

	logger.Info("service started")
}

func boolPtr(v bool) *bool { return &v }
```

## Loki behavior

- Loki is optional.
- If `Loki` config is omitted and `LOKI_URL` is unset, only console logging is used.
- Delivery is fire-and-forget and never blocks request handling.
- `debug` logs are not shipped to Loki (console only), matching `log-lib` behavior.
