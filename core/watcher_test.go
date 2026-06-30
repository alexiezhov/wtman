package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDirSnapshot_ignoresHidden(t *testing.T) {
	dir := tempDir(t)
	if err := os.MkdirAll(filepath.Join(dir, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, ".hidden"), "x")
	writeFile(t, filepath.Join(dir, "b"), "x")

	snap := dirSnapshot(dir)
	if snap != "a\x00b" {
		t.Errorf("dirSnapshot = %q, want a<nul>b", snap)
	}
}

func TestDirSnapshot_missingDir(t *testing.T) {
	if snap := dirSnapshot(filepath.Join(tempDir(t), "nope")); snap != "" {
		t.Errorf("missing dir snapshot = %q, want empty", snap)
	}
}

func TestDirWatcher_emitsOnChange(t *testing.T) {
	source := tempDir(t)
	target := tempDir(t)
	w := NewDirWatcher(source, target, 20*time.Millisecond)
	w.Start()
	defer w.Stop()

	// Let the poller capture its baseline snapshot before we change anything,
	// otherwise the change may be present in the very first snapshot.
	time.Sleep(100 * time.Millisecond)

	// Create an entry in the target dir; expect a TargetChanged event.
	if err := os.MkdirAll(filepath.Join(target, "feat"), 0o755); err != nil {
		t.Fatal(err)
	}

	select {
	case ev := <-w.Events():
		if ev.Kind != TargetChanged {
			t.Errorf("event kind = %v, want TargetChanged", ev.Kind)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watch event")
	}
}

func TestDirWatcher_sourceChange(t *testing.T) {
	source := tempDir(t)
	target := tempDir(t)
	w := NewDirWatcher(source, target, 20*time.Millisecond)
	w.Start()
	defer w.Stop()

	time.Sleep(100 * time.Millisecond)

	if err := os.MkdirAll(filepath.Join(source, "newrepo"), 0o755); err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-w.Events():
		if ev.Kind != SourceChanged {
			t.Errorf("event kind = %v, want SourceChanged", ev.Kind)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for source watch event")
	}
}
