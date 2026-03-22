package logger

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorGrey   = "\033[90m"
)

type ColoredFormatter struct{}

func (f *ColoredFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := colorBlue + entry.Time.Format("15:04:05") + colorReset

	var levelColored string
	switch entry.Level {
	case logrus.InfoLevel:
		levelColored = colorGreen + "info" + colorReset
	case logrus.WarnLevel:
		levelColored = colorYellow + "warn" + colorReset
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColored = colorRed + "error" + colorReset
	case logrus.DebugLevel, logrus.TraceLevel:
		levelColored = colorGrey + "debug" + colorReset
	default:
		levelColored = entry.Level.String()
	}

	msg := entry.Message
	if entry.Level == logrus.DebugLevel || entry.Level == logrus.TraceLevel {
		msg = colorGrey + msg + colorReset
	}

	meta := formatMeta(entry.Data)
	line := fmt.Sprintf("%s [%s]: %s", timestamp, levelColored, msg)
	if meta != "" {
		line += " " + meta
	}
	line += "\n"
	return []byte(line), nil
}

func formatMeta(data logrus.Fields) string {
	if len(data) == 0 {
		return ""
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s%s%s=%s%v%s", colorPurple, k, colorReset, colorPurple, data[k], colorReset))
	}
	return strings.Join(parts, " ")
}
