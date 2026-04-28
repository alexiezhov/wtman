package tui

import "github.com/hibobio/wtman/core"

// Outcome messages emitted by sub-models

type CommandMsg struct {
	Name string
	Arg  string
}

type BranchSelectedMsg struct {
	Branch core.FeatureBranch
}

type ReposConfirmedMsg struct {
	Repos []core.RepoEntry
}

type ReposCancelledMsg struct{}

type PromptResultMsg struct {
	Value     string
	Cancelled bool
}

type ConfirmResultMsg struct {
	Confirmed bool
}

type DirtyWorktreesMsg struct {
	DirtyRepos []string
}

type DirtyDeleteMsg struct {
	DirtyRepos []string
}

type clearErrorMsg struct{}

// Operation result messages

type OperationDoneMsg struct {
	Err error
}

// Watcher messages

type WatchEventMsg struct {
	Event core.WatchEvent
}
