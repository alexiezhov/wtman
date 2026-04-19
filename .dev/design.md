# wtman -- Design Principles & Screens

## Design Principles

1. **Core / TUI separation** -- `core/` is pure Go with zero TUI dependencies. All git, filesystem, and config logic lives here. The TUI is one possible frontend; another (CLI, web, tests) could use `core/` directly.

2. **Sub-models as self-contained widgets** -- each TUI component (branch list, repo select, status bar, prompt) is an independent bubbletea model. It accepts input when focused and emits typed outcome messages (`ReposSelectedMsg`, `CommandMsg`, etc.). It knows nothing about who owns it or what mode the app is in.

3. **App as orchestrator** -- `app.go` owns layout, focus, and mode transitions. It routes key messages to the focused widget, reacts to outcome messages, and dispatches core operations. Changing the UI layout (e.g. from sequential modes to side-by-side panels with Tab/Shift+Tab) only requires changes to `app.go`.

4. **Blocking UI during operations** -- long-running git operations run as background `tea.Cmd`s. While running, a spinner is shown in the status bar and all input is blocked. No race conditions, no queuing.

5. **Polling for freshness** -- source and target directories are polled every 2 seconds. Changes (new repos, new/removed feature branches) are reflected in the UI automatically.

6. **Sorted repo names everywhere** -- repo names are always sorted alphabetically: in the repo select list, in the branch list Repos column, and in prompt summaries.

## Screens

### 1. Branch List (default)

```
┌─────────────────────────────────────────────────────────────────┐
│  wtman                                                          │
│                                                                 │
│  Date       │ Branch                  │ Repos                   │
│  ───────────┼─────────────────────────┼──────────────────────── │
│  2026-01-01 │ rename-report-fields    │ billing, report-engine  │
│░░2026-03-15░│░migrate-auth-service░░░░│░auth, billing, paym...░░│
│  2026-04-10 │ fix-payment-rounding    │ payment-gateway         │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│  up/down navigate  ENTER update  / command                      │
└─────────────────────────────────────────────────────────────────┘
```

`░` = highlighted background on the focused row (full terminal width).

### 2. Branch List with Command Bar

```
┌─────────────────────────────────────────────────────────────────┐
│  wtman                                                          │
│                                                                 │
│  Date       │ Branch                  │ Repos                   │
│  ───────────┼─────────────────────────┼──────────────────────── │
│  2026-01-01 │ rename-report-fields    │ billing, report-engine  │
│░░2026-03-15░│░migrate-auth-service░░░░│░auth, billing, paym...░░│
│  2026-04-10 │ fix-payment-rounding    │ payment-gateway         │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│  /ne▌                                                           │
│   /new         /rename        /delete                           │
└─────────────────────────────────────────────────────────────────┘
```

Autocomplete uses fuzzy matching (same algorithm as repo filter). Suggestions shown dimmed below the input; non-matching commands disappear as you type. Ranked by match quality.

### 3. Repo Select (via /new)

```
┌─────────────────────────────────────────────────────────────────┐
│  wtman ── new feature branch                                    │
│                                                                 │
│  [ ] auth-service                                               │
│  [ ] billing-api                                                │
│░░[x] payment-gateway░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│  [x] report-engine                                              │
│  [ ] scheduler                                                  │
│  [ ] user-management                                            │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│  up/down navigate  SPACE toggle  type to filter  ESC cancel     │
│  ENTER confirm (2)                                              │
└─────────────────────────────────────────────────────────────────┘
```

`[x]` rendered in success color. Selected count shown next to ENTER confirm.

### 4. Repo Select with Filter

```
┌─────────────────────────────────────────────────────────────────┐
│  wtman ── new feature branch                                    │
│                                                                 │
│░░[x] payment-gateway░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│  [x] report-engine                                              │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│  Filter: re▌                                                    │
│  up/down navigate  SPACE toggle  ESC clear filter               │
│  ENTER confirm (2)                                              │
└─────────────────────────────────────────────────────────────────┘
```

### 5. Repo Select (update existing branch)

Same layout as screen 3, but repos already in the feature branch are pre-selected. Header reads `wtman ── update <branch-name>`. Toggling off a repo will remove its worktree on confirm. If any unchecked repos have uncommitted changes, a dirty-worktree confirmation is shown before proceeding.

### 5b. Dirty Worktree Confirmation (on update with dirty repos)

```
┌─────────────────────────────────────────────────────────────────┐
│  wtman                                                          │
│                                                                 │
│  Date       │ Branch                  │ Repos                   │
│  ───────────┼─────────────────────────┼──────────────────────── │
│  2026-01-01 │ rename-report-fields    │ billing, report-engine  │
│░░2026-03-15░│░migrate-auth-service░░░░│░auth, billing, paym...░░│
│  2026-04-10 │ fix-payment-rounding    │ payment-gateway         │
│                                                                 │
│                                                                 │
│  Dirty worktrees: auth, billing. Force remove? (uncommitted     │
│  changes will be lost)                                          │
│  y confirm  n/ESC cancel                                        │
└─────────────────────────────────────────────────────────────────┘
```

### 6. Branch Name Prompt

```
┌─────────────────────────────────────────────────────────────────┐
│  wtman ── new feature branch                                    │
│                                                                 │
│  Selected: payment-gateway, report-engine                       │
│                                                                 │
│  Branch name: my-new-feature▌                                   │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│  ENTER create  ESC back                                         │
└─────────────────────────────────────────────────────────────────┘
```

### 7. Delete Confirmation

```
┌─────────────────────────────────────────────────────────────────┐
│  wtman                                                          │
│                                                                 │
│  Date       │ Branch                  │ Repos                   │
│  ───────────┼─────────────────────────┼──────────────────────── │
│  2026-01-01 │ rename-report-fields    │ billing, report-engine  │
│░░2026-03-15░│░migrate-auth-service░░░░│░auth, billing, paym...░░│
│  2026-04-10 │ fix-payment-rounding    │ payment-gateway         │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│  Delete "migrate-auth-service"? Removes worktrees & branches.   │
│  y confirm  n/ESC cancel                                        │
└─────────────────────────────────────────────────────────────────┘
```

### 8. Rename Prompt

```
┌─────────────────────────────────────────────────────────────────┐
│  wtman                                                          │
│                                                                 │
│  Date       │ Branch                  │ Repos                   │
│  ───────────┼─────────────────────────┼──────────────────────── │
│  2026-01-01 │ rename-report-fields    │ billing, report-engine  │
│░░2026-03-15░│░migrate-auth-service░░░░│░auth, billing, paym...░░│
│  2026-04-10 │ fix-payment-rounding    │ payment-gateway         │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│  Rename to: migrate-auth-v2▌                                    │
│  ENTER rename  ESC cancel                                       │
└─────────────────────────────────────────────────────────────────┘
```

### 9. Spinner (during operations)

```
┌─────────────────────────────────────────────────────────────────┐
│  wtman                                                          │
│                                                                 │
│  Date       │ Branch                  │ Repos                   │
│  ───────────┼─────────────────────────┼──────────────────────── │
│  2026-01-01 │ rename-report-fields    │ billing, report-engine  │
│░░2026-03-15░│░migrate-auth-service░░░░│░auth, billing, paym...░░│
│  2026-04-10 │ fix-payment-rounding    │ payment-gateway         │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
│  ⣾ Creating worktrees...                                        │
└─────────────────────────────────────────────────────────────────┘
```

All input blocked. Spinner auto-animates via bubbletea tick commands.

### 10. Error Display

Errors from any operation are shown between the repo list and the status bar/hint area, styled in the error color. Errors auto-dismiss after 5 seconds. No errors are swallowed -- all git failures, config save errors, workspace file errors, and post-command errors are surfaced.

```
┌─────────────────────────────────────────────────────────────────┐
│  wtman                                                          │
│                                                                 │
│  Date       │ Branch                  │ Repos                   │
│  ───────────┼─────────────────────────┼──────────────────────── │
│  2026-01-01 │ rename-report-fields    │ billing, report-engine  │
│░░2026-03-15░│░migrate-auth-service░░░░│░auth, billing, paym...░░│
│  2026-04-10 │ fix-payment-rounding    │ payment-gateway         │
│                                                                 │
│  Error: failed repos:                                           │
│    auth: branch already checked out                             │
│                                                                 │
│  up/down navigate  ENTER update  / command                      │
└─────────────────────────────────────────────────────────────────┘
```
