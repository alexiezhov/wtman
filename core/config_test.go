package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_missingFileReturnsDefaults(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(tempDir(t), "nope.json"))
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	def := DefaultConfig()
	if cfg.PostCommand != def.PostCommand {
		t.Errorf("PostCommand = %q, want default %q", cfg.PostCommand, def.PostCommand)
	}
	if cfg.ScanDepth != def.ScanDepth {
		t.Errorf("ScanDepth = %d, want %d", cfg.ScanDepth, def.ScanDepth)
	}
	if cfg.LogLevel != def.LogLevel {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, def.LogLevel)
	}
}

func TestLoadConfig_parsesAndClampsScanDepth(t *testing.T) {
	path := filepath.Join(tempDir(t), "config.json")
	writeFile(t, path, `{"source_dir":"/s","target_dir":"/t","scan_depth":0}`)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourceDir != "/s" || cfg.TargetDir != "/t" {
		t.Errorf("dirs = %q,%q", cfg.SourceDir, cfg.TargetDir)
	}
	if cfg.ScanDepth != 1 {
		t.Errorf("ScanDepth = %d, want clamped to 1", cfg.ScanDepth)
	}
}

func TestLoadConfig_invalidJSON(t *testing.T) {
	path := filepath.Join(tempDir(t), "bad.json")
	writeFile(t, path, `{not json`)
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadConfig_mergesPartialColors(t *testing.T) {
	path := filepath.Join(tempDir(t), "config.json")
	writeFile(t, path, `{"colors":{"title":"42"}}`)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Colors.Title != "42" {
		t.Errorf("Title = %q, want overridden 42", cfg.Colors.Title)
	}
	if cfg.Colors.Success != DefaultColors().Success {
		t.Errorf("Success = %q, want default %q", cfg.Colors.Success, DefaultColors().Success)
	}
}

func TestSaveConfig_roundTrip(t *testing.T) {
	path := filepath.Join(tempDir(t), "sub", "config.json")
	in := DefaultConfig()
	in.SourceDir = "/repos"
	in.TargetDir = "/branches"

	if err := SaveConfig(path, in); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	out, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if out.SourceDir != in.SourceDir || out.TargetDir != in.TargetDir {
		t.Errorf("round trip mismatch: %+v vs %+v", out, in)
	}

	// Ensure it is valid, indented JSON on disk.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var generic map[string]any
	if err := json.Unmarshal(data, &generic); err != nil {
		t.Errorf("saved config is not valid JSON: %v", err)
	}
}

func TestMergeColors_emptyUsesDefaults(t *testing.T) {
	merged := mergeColors(ColorsConfig{}, DefaultColors())
	if merged != DefaultColors() {
		t.Errorf("mergeColors of empty = %+v, want defaults", merged)
	}
}
