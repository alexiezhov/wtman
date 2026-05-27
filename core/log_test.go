package core

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		in      string
		want    slog.Level
		wantErr bool
	}{
		{"", slog.LevelInfo, false},
		{"info", slog.LevelInfo, false},
		{"DEBUG", slog.LevelDebug, false},
		{"warn", slog.LevelWarn, false},
		{"error", slog.LevelError, false},
		{"off", levelOff, false},
		{"disabled", levelOff, false},
		{"verbose", 0, true},
	}
	for _, tc := range tests {
		got, err := ParseLogLevel(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseLogLevel(%q): want error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseLogLevel(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseLogLevel(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestInitLogger_respectsLevel(t *testing.T) {
	var buf bytes.Buffer
	InitLogger(slog.LevelWarn, &buf)

	slog.Debug("hidden")
	slog.Info("hidden")
	slog.Warn("shown")
	slog.Error("shown")

	out := buf.String()
	if strings.Contains(out, "hidden") {
		t.Fatalf("expected warn/error only, got:\n%s", out)
	}
	if !strings.Contains(out, "shown") {
		t.Fatalf("expected warn/error logs, got:\n%s", out)
	}
}

func TestInitLogger_off(t *testing.T) {
	var buf bytes.Buffer
	InitLogger(levelOff, &buf)

	slog.Error("still hidden")
	if buf.Len() != 0 {
		t.Fatalf("expected no output at off level, got %q", buf.String())
	}
}
