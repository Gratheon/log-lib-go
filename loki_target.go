package logger

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type lokiTarget struct {
	config LokiConfig
	client *http.Client
}

func newLokiTarget(config LokiConfig) *lokiTarget {
	return &lokiTarget{
		config: config,
		client: &http.Client{Timeout: 2 * time.Second},
	}
}

func (l *lokiTarget) Levels() []logrus.Level {
	return []logrus.Level{logrus.InfoLevel, logrus.WarnLevel, logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel}
}

func (l *lokiTarget) Fire(entry *logrus.Entry) error {
	payload, err := l.buildPayload(entry)
	if err != nil {
		return nil
	}

	go l.push(payload)
	return nil
}

func (l *lokiTarget) buildPayload(entry *logrus.Entry) ([]byte, error) {
	meta := map[string]interface{}{}
	for k, v := range entry.Data {
		meta[k] = v
	}

	line := map[string]interface{}{
		"timestamp": entry.Time.UTC().Format(time.RFC3339Nano),
		"level":     entry.Level.String(),
		"service":   l.config.Service,
		"message":   strings.TrimSpace(entry.Message),
		"meta":      meta,
	}

	if stack, ok := entry.Data["stack"]; ok {
		line["stacktrace"] = stack
	}

	lineJSON, err := json.Marshal(line)
	if err != nil {
		return nil, err
	}

	ns := fmt.Sprintf("%d", entry.Time.UnixNano())
	body := map[string]interface{}{
		"streams": []interface{}{
			map[string]interface{}{
				"stream": l.config.Labels,
				"values": []interface{}{
					[]string{ns, string(lineJSON)},
				},
			},
		},
	}

	return json.Marshal(body)
}

func (l *lokiTarget) push(payload []byte) {
	req, err := http.NewRequest(http.MethodPost, l.config.URL, bytes.NewReader(payload))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if l.config.TenantID != "" {
		req.Header.Set("X-Scope-OrgID", l.config.TenantID)
	}
	if l.config.Username != "" || l.config.Password != "" {
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(l.config.Username+":"+l.config.Password)))
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}
