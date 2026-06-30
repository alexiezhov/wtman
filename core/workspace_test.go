package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceFileName(t *testing.T) {
	if got := WorkspaceFileName("/b/a--feat--add-field"); got != "a--feat--add-field.code-workspace" {
		t.Errorf("WorkspaceFileName = %q", got)
	}
}

func TestCreateCursorWorkspace(t *testing.T) {
	branchDir := filepath.Join(tempDir(t), "feat--x")
	if err := os.MkdirAll(branchDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Pass repos out of order; the workspace must store them sorted.
	if err := CreateCursorWorkspace(branchDir, []string{"report-engine", "billing-api"}); err != nil {
		t.Fatalf("CreateCursorWorkspace: %v", err)
	}

	path := filepath.Join(branchDir, "feat--x.code-workspace")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("workspace file not written: %v", err)
	}

	var ws struct {
		Folders  []struct{ Path string } `json:"folders"`
		Settings map[string]any          `json:"settings"`
	}
	if err := json.Unmarshal(data, &ws); err != nil {
		t.Fatalf("invalid workspace JSON: %v", err)
	}
	if len(ws.Folders) != 2 {
		t.Fatalf("expected 2 folders, got %d", len(ws.Folders))
	}
	if ws.Folders[0].Path != "billing-api" || ws.Folders[1].Path != "report-engine" {
		t.Errorf("folders not sorted: %+v", ws.Folders)
	}
	if ws.Settings == nil {
		t.Error("settings should be present (empty object), got null")
	}
}

func TestCreateCursorWorkspace_empty(t *testing.T) {
	branchDir := filepath.Join(tempDir(t), "feat")
	if err := os.MkdirAll(branchDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CreateCursorWorkspace(branchDir, nil); err != nil {
		t.Fatalf("empty workspace: %v", err)
	}
	if _, err := os.Stat(filepath.Join(branchDir, "feat.code-workspace")); err != nil {
		t.Errorf("workspace file missing: %v", err)
	}
}
