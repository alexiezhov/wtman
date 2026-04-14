package core

import (
	"os"
	"sort"
	"strings"
	"time"
)

type WatchEventKind int

const (
	SourceChanged WatchEventKind = iota
	TargetChanged
)

type WatchEvent struct {
	Kind WatchEventKind
}

type DirWatcher struct {
	sourceDir string
	targetDir string
	interval  time.Duration
	events    chan WatchEvent
	stop      chan struct{}
}

func NewDirWatcher(sourceDir, targetDir string, interval time.Duration) *DirWatcher {
	return &DirWatcher{
		sourceDir: sourceDir,
		targetDir: targetDir,
		interval:  interval,
		events:    make(chan WatchEvent, 4),
		stop:      make(chan struct{}),
	}
}

func (w *DirWatcher) Events() <-chan WatchEvent {
	return w.events
}

func (w *DirWatcher) SetSourceDir(dir string) {
	w.sourceDir = dir
}

func (w *DirWatcher) SetTargetDir(dir string) {
	w.targetDir = dir
}

func (w *DirWatcher) Start() {
	go w.poll()
}

func (w *DirWatcher) Stop() {
	close(w.stop)
}

func (w *DirWatcher) poll() {
	prevSource := dirSnapshot(w.sourceDir)
	prevTarget := dirSnapshot(w.targetDir)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			curSource := dirSnapshot(w.sourceDir)
			if curSource != prevSource {
				prevSource = curSource
				select {
				case w.events <- WatchEvent{Kind: SourceChanged}:
				default:
				}
			}

			curTarget := dirSnapshot(w.targetDir)
			if curTarget != prevTarget {
				prevTarget = curTarget
				select {
				case w.events <- WatchEvent{Kind: TargetChanged}:
				default:
				}
			}
		}
	}
}

func dirSnapshot(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var names []string
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return strings.Join(names, "\x00")
}
