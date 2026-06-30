package cli

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexiezhov/wtman/core"
)

// --- helpers ---

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func tempDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func initRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "init", "-q")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "wtman test")
	git(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "initial")
	git(t, dir, "branch", "-M", "main")
}

// captureStdout runs f with os.Stdout redirected to a pipe and returns what was written.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(r)
		done <- string(data)
	}()
	f()
	w.Close()
	os.Stdout = old
	return <-done
}

func testConfig(t *testing.T) (core.Config, string, string) {
	t.Helper()
	source := tempDir(t)
	target := tempDir(t)
	return core.Config{
		SourceDir: source,
		TargetDir: target,
		ScanDepth: 1,
	}, source, target
}

// --- pure-function tests ---

func TestParseInterspersed(t *testing.T) {
	cases := []struct {
		name           string
		args           []string
		wantPositional []string
		wantNoHook     bool
		wantFrom       string
	}{
		{"flags after positionals", []string{"feat", "a,b", "-n", "--from", "dev"}, []string{"feat", "a,b"}, true, "dev"},
		{"flags before positionals", []string{"-n", "--from", "dev", "feat", "a,b"}, []string{"feat", "a,b"}, true, "dev"},
		{"interspersed", []string{"feat", "-n", "a,b", "--from", "dev"}, []string{"feat", "a,b"}, true, "dev"},
		{"no flags", []string{"feat", "a,b"}, []string{"feat", "a,b"}, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := flag.NewFlagSet("new", flag.ContinueOnError)
			noHook := fs.Bool("n", false, "")
			from := fs.String("from", "", "")
			got := parseInterspersed(fs, tc.args)
			if strings.Join(got, ",") != strings.Join(tc.wantPositional, ",") {
				t.Errorf("positional = %v, want %v", got, tc.wantPositional)
			}
			if *noHook != tc.wantNoHook {
				t.Errorf("noHook = %v, want %v", *noHook, tc.wantNoHook)
			}
			if *from != tc.wantFrom {
				t.Errorf("from = %q, want %q", *from, tc.wantFrom)
			}
		})
	}
}

func TestFilterRepos(t *testing.T) {
	all := []core.RepoEntry{
		{Name: "auth", Path: "/s/auth"},
		{Name: "billing", Path: "/s/billing"},
		{Name: "payments", Path: "/s/payments"},
	}
	got := filterRepos(all, []string{"billing", " auth "})
	if len(got) != 2 || got[0].Name != "billing" || got[1].Name != "auth" {
		t.Errorf("filterRepos = %v, want [billing auth] preserving requested order", got)
	}
}

// --- command happy-path tests (JSON on stdout) ---

func TestRun_repos(t *testing.T) {
	requireGit(t)
	cfg, source, _ := testConfig(t)
	initRepo(t, filepath.Join(source, "auth"))
	initRepo(t, filepath.Join(source, "billing"))

	out := captureStdout(t, func() { Run(cfg, []string{"repos"}) })

	var repos []struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(out), &repos); err != nil {
		t.Fatalf("invalid JSON %q: %v", out, err)
	}
	if len(repos) != 2 || repos[0].Name != "auth" || repos[1].Name != "billing" {
		t.Errorf("repos = %+v", repos)
	}
}

func TestRun_newLsUpdateRm(t *testing.T) {
	requireGit(t)
	cfg, source, target := testConfig(t)
	initRepo(t, filepath.Join(source, "auth"))
	initRepo(t, filepath.Join(source, "billing"))

	// new (skip post hook with -n)
	out := captureStdout(t, func() {
		Run(cfg, []string{"new", "feat/x", "auth", "-n"})
	})
	var created struct {
		Branch string   `json:"branch"`
		Path   string   `json:"path"`
		Repos  []string `json:"repos"`
		Base   string   `json:"base"`
	}
	if err := json.Unmarshal([]byte(out), &created); err != nil {
		t.Fatalf("new JSON %q: %v", out, err)
	}
	if created.Branch != "feat/x" || strings.Join(created.Repos, ",") != "auth" {
		t.Errorf("new output = %+v", created)
	}
	// Workspace file is named after the encoded branch dir.
	if _, err := os.Stat(filepath.Join(target, "feat--x", "feat--x.code-workspace")); err != nil {
		t.Errorf("workspace file missing: %v", err)
	}

	// ls shows the branch
	out = captureStdout(t, func() { Run(cfg, []string{"ls"}) })
	var listed []struct {
		Name  string   `json:"name"`
		Repos []string `json:"repos"`
		Date  string   `json:"date"`
	}
	if err := json.Unmarshal([]byte(out), &listed); err != nil {
		t.Fatalf("ls JSON %q: %v", out, err)
	}
	if len(listed) != 1 || listed[0].Name != "feat/x" {
		t.Fatalf("ls = %+v", listed)
	}

	// update to add billing
	out = captureStdout(t, func() {
		Run(cfg, []string{"update", "feat/x", "auth,billing"})
	})
	if !strings.Contains(out, `"ok":true`) {
		t.Errorf("update output = %q", out)
	}
	if on := core.ListReposOnDisk(filepath.Join(target, "feat--x")); strings.Join(on, ",") != "auth,billing" {
		t.Errorf("after update repos = %v", on)
	}

	// rm removes it
	out = captureStdout(t, func() { Run(cfg, []string{"rm", "feat/x"}) })
	if !strings.Contains(out, `"ok":true`) {
		t.Errorf("rm output = %q", out)
	}
	if _, err := os.Stat(filepath.Join(target, "feat--x")); !os.IsNotExist(err) {
		t.Errorf("branch dir should be gone, err=%v", err)
	}
}

func TestRun_newFromBase(t *testing.T) {
	requireGit(t)
	cfg, source, target := testConfig(t)
	repo := filepath.Join(source, "auth")
	initRepo(t, repo)
	git(t, repo, "checkout", "-q", "-b", "develop")
	git(t, repo, "checkout", "-q", "main")

	out := captureStdout(t, func() {
		Run(cfg, []string{"new", "feat", "auth", "-n", "--from", "develop"})
	})
	var created struct {
		Base string `json:"base"`
	}
	if err := json.Unmarshal([]byte(out), &created); err != nil {
		t.Fatalf("new JSON %q: %v", out, err)
	}
	if created.Base != "develop" {
		t.Errorf("base = %q, want develop", created.Base)
	}
	if _, err := os.Stat(filepath.Join(target, "feat", "auth")); err != nil {
		t.Errorf("worktree missing: %v", err)
	}
}

// TestRun_update_dirtyWithoutForce verifies that removing a dirty worktree
// without -f reports {"error":"dirty"} and exits non-zero. cmdUpdate calls
// os.Exit on that path, so the scenario runs in a re-executed subprocess that
// builds its own state (state is not shared across the re-exec).
func TestRun_update_dirtyWithoutForce(t *testing.T) {
	requireGit(t)

	if os.Getenv("WTMAN_DIRTY_SUBPROC") == "1" {
		cfg, source, target := testConfig(t)
		initRepo(t, filepath.Join(source, "auth"))
		initRepo(t, filepath.Join(source, "billing"))
		Run(cfg, []string{"new", "feat", "auth,billing", "-n"})
		// Dirty the auth worktree, then try to drop it without -f.
		if err := os.WriteFile(filepath.Join(target, "feat", "auth", "README.md"), []byte("dirty\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		Run(cfg, []string{"update", "feat", "billing"})
		return // unreachable: Run os.Exit(1)s on the dirty path
	}

	cmd := exec.Command(os.Args[0], "-test.run", "^TestRun_update_dirtyWithoutForce$", "-test.v")
	cmd.Env = append(os.Environ(), "WTMAN_DIRTY_SUBPROC=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit for dirty update, output:\n%s", out)
	}
	if !strings.Contains(string(out), `"error":"dirty"`) {
		t.Errorf("expected dirty JSON in output, got:\n%s", out)
	}
}

func TestRun_mv(t *testing.T) {
	requireGit(t)
	cfg, source, target := testConfig(t)
	initRepo(t, filepath.Join(source, "auth"))
	captureStdout(t, func() { Run(cfg, []string{"new", "old", "auth", "-n"}) })

	out := captureStdout(t, func() { Run(cfg, []string{"mv", "old", "new"}) })
	if !strings.Contains(out, `"ok":true`) {
		t.Errorf("mv output = %q", out)
	}
	if _, err := os.Stat(filepath.Join(target, "new")); err != nil {
		t.Errorf("renamed dir missing: %v", err)
	}
}
