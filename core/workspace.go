package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

type workspaceFolder struct {
	Path string `json:"path"`
}

type workspaceFile struct {
	Folders  []workspaceFolder      `json:"folders"`
	Settings map[string]interface{} `json:"settings"`
}

func CreateCursorWorkspace(branchDir string, repoNames []string) error {
	sorted := make([]string, len(repoNames))
	copy(sorted, repoNames)
	sort.Strings(sorted)

	folders := make([]workspaceFolder, len(sorted))
	for i, name := range sorted {
		folders[i] = workspaceFolder{Path: name}
	}

	ws := workspaceFile{
		Folders:  folders,
		Settings: map[string]interface{}{},
	}

	data, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(branchDir, "workspace.code-workspace")
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
