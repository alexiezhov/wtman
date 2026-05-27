package core

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
	LogLevelOff   = "off"
)

// levelOff is above slog.LevelError so nothing is emitted when log_level is "off".
const levelOff = slog.Level(32)

// ParseLogLevel maps a config/flag string to a slog level.
func ParseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", LogLevelInfo:
		return slog.LevelInfo, nil
	case LogLevelDebug:
		return slog.LevelDebug, nil
	case LogLevelWarn:
		return slog.LevelWarn, nil
	case LogLevelError:
		return slog.LevelError, nil
	case LogLevelOff, "none", "disabled":
		return levelOff, nil
	default:
		return 0, fmt.Errorf("invalid log level %q (want debug, info, warn, error, or off)", s)
	}
}

// NormalizeLogLevel returns the canonical level name or defaultLogLevel if empty.
func NormalizeLogLevel(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case LogLevelDebug, LogLevelWarn, LogLevelError, LogLevelOff, "none", "disabled":
		return strings.ToLower(strings.TrimSpace(s))
	case "", LogLevelInfo:
		return LogLevelInfo
	default:
		return strings.ToLower(strings.TrimSpace(s))
	}
}

// InitLogger configures the default slog logger to write text logs to w (stderr by default).
func InitLogger(level slog.Level, w io.Writer) {
	if w == nil {
		w = os.Stderr
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})))
}
