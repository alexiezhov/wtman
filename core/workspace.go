package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type workspaceFolder struct {
	Path string `json:"path"`
}

type workspaceFile struct {
	Folders  []workspaceFolder      `json:"folders"`
	Settings map[string]interface{} `json:"settings"`
}

// WorkspaceFileName returns the .code-workspace filename for a branch directory,
// derived from the encoded branch directory name (e.g. a--feat--add-field.code-workspace).
func WorkspaceFileName(branchDir string) string {
	return filepath.Base(branchDir) + ".code-workspace"
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

	path := filepath.Join(branchDir, WorkspaceFileName(branchDir))
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// CreateIntellijWorkspace writes a top-level .idea/ project at branchDir that
// references each repo as an IntelliJ module. Where a repo already ships its own
// .idea/<name>.iml, modules.xml points at that file directly so IntelliJ also
// surfaces the run configurations the repo has committed under its own .idea/.
// Where a repo has no .iml, a minimal fallback .iml is generated under the
// top-level .idea/.
func CreateIntellijWorkspace(branchDir string, repoNames []string) error {
	sorted := make([]string, len(repoNames))
	copy(sorted, repoNames)
	sort.Strings(sorted)

	ideaDir := filepath.Join(branchDir, ".idea")
	if err := os.MkdirAll(ideaDir, 0o755); err != nil {
		return err
	}

	branchName := filepath.Base(branchDir)
	if err := os.WriteFile(filepath.Join(ideaDir, ".name"), []byte(branchName+"\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(ideaDir, ".gitignore"), []byte("workspace.xml\nshelf/\n"), 0o644); err != nil {
		return err
	}

	var moduleEntries []string
	for _, name := range sorted {
		imlRel, err := resolveOrCreateIml(ideaDir, branchDir, name)
		if err != nil {
			return err
		}
		entry := fmt.Sprintf(
			`      <module fileurl="file://$PROJECT_DIR$/%s" filepath="$PROJECT_DIR$/%s" />`,
			imlRel, imlRel,
		)
		moduleEntries = append(moduleEntries, entry)
	}

	modulesXML := `<?xml version="1.0" encoding="UTF-8"?>
<project version="4">
  <component name="ProjectModuleManager">
    <modules>
` + strings.Join(moduleEntries, "\n") + `
    </modules>
  </component>
</project>
`
	return os.WriteFile(filepath.Join(ideaDir, "modules.xml"), []byte(modulesXML), 0o644)
}

// resolveOrCreateIml returns the .iml path (relative to branchDir, forward
// slashes for IntelliJ's $PROJECT_DIR$) for a repo. If the repo has its own
// .idea/*.iml, that path is returned. Otherwise a fallback .iml is generated
// under the top-level .idea/.
func resolveOrCreateIml(ideaDir, branchDir, repoName string) (string, error) {
	repoIdea := filepath.Join(branchDir, repoName, ".idea")
	matches, _ := filepath.Glob(filepath.Join(repoIdea, "*.iml"))
	if len(matches) > 0 {
		return repoName + "/.idea/" + filepath.Base(matches[0]), nil
	}

	imlName := repoName + ".iml"
	iml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<module type="WEB_MODULE" version="4">
  <component name="NewModuleRootManager" inherit-compiler-output="true">
    <exclude-output />
    <content url="file://$MODULE_DIR$/../%s" />
    <orderEntry type="inheritedJdk" />
    <orderEntry type="sourceFolder" forTests="false" />
  </component>
</module>
`, repoName)
	if err := os.WriteFile(filepath.Join(ideaDir, imlName), []byte(iml), 0o644); err != nil {
		return "", err
	}
	return ".idea/" + imlName, nil
}
