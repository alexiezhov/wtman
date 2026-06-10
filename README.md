# wtman

A multi-repo git worktree manager with an interactive TUI and CLI. Creates feature branches across multiple repositories simultaneously, manages worktrees under a shared directory, and generates Cursor/VS Code multi-root workspace files.

## Prerequisites

```bash
brew install go git
```

- **Go 1.26+** — [install via Homebrew](https://formulae.brew.sh/formula/go) or from [go.dev](https://go.dev/dl/)
- **git** — required at runtime for all worktree operations
- **macOS** — the approval watcher uses `osascript` and `script` (both ship with macOS)

## Build

```bash
go build -o wtman .
```

## Install

Copy the binary somewhere on your `PATH`:

```bash
go build -o wtman . && mv wtman ~/.local/bin/
```

Or install directly:

```bash
go install github.com/hibobio/wtman@latest
```

## Configuration

Config file: `~/.config/wtman/config.json`

```json
{
  "source_dir": "/path/to/repos",
  "target_dir": "/path/to/branches",
  "post_command": "agent {{dir}}",
  "scan_depth": 1,
  "log_level": "info"
}
```

- `source_dir` — directory containing your git repositories
- `target_dir` — directory where feature branch worktrees are created
- `log_level` — logging verbosity: `debug`, `info`, `warn`, `error`, or `off` (default `info`). Logs are written to stderr.
- `post_command` — shell command run after worktrees are created; `{{dir}}` is replaced with the branch directory path, and `{{workspace}}` with the absolute path to the generated `.code-workspace` file. For example:
  - `"open {{workspace}}"` - opens workspace in Cursor IDE
  - `"tmux split-window -h \"zsh -c 'cd {{dir}} && tmux select-pane -T \\\"${PWD##*/}\\\"; cursor-watcher agent; exec zsh'\""` - runs Cursor CLI with Cursor Approval Watcher (see below) in a new tmux pane
- `scan_depth` — how deep to look for git repos in source dir

## Usage

Run without arguments to launch the interactive TUI:

```bash
wtman
```

CLI commands for scripting:

```bash
wtman ls                                    # List feature branches (JSON)
wtman repos                                 # List available source repos (JSON)
wtman new <branch> <repos> [-n] [--from <ref>]  # Create branch with worktrees (-n skips post hook; --from sets the base ref)
wtman rm <branch> [-f]                      # Delete branch (-f force even if dirty)
wtman update <branch> <repos> [-f]          # Set repos for branch (-f force dirty removal)
wtman mv <old> <new>                        # Rename branch
wtman pull                                  # Pull all repos under source_dir
```

By default, new branches are created from each repo's `main`/`master`. Pass `--from <ref>` to base them on another branch, tag, or commit instead (resolved per repo as a local ref, then `origin/<ref>`):

```bash
wtman new my-feature auth,billing --from develop
```

Flags (`-n`, `--from`, `-f`) may appear before, after, or interspersed with the positional arguments.

Enable verbose logging:

```bash
wtman -v ls                       # debug logs on stderr
wtman --log-level warn new feat-x auth,billing
```

---


## Cursor Approval Watcher

`watcher.sh` is a wrapper script that monitors a command's output for Cursor approval prompts and sends a macOS notification when one is detected.

When running Cursor in agent/CLI mode, it sometimes pauses to ask "Run this command?" before executing shell commands. If you're working in another window, you won't notice the prompt. The watcher solves this by sending a desktop notification so you can switch back and approve it.

### Install the watcher

Copy `watcher.sh` somewhere on your `PATH`:

```bash
cp watcher.sh ~/.local/bin/cursor-watcher
chmod +x ~/.local/bin/cursor-watcher
```

### Usage

Wrap any command that might produce Cursor approval prompts:

```bash
cursor-watcher agent
```

Or use it with any long-running command where Cursor is operating:

```bash
cursor-watcher <command> [args...]
```

When the output contains a line matching "Run this command?" (case-insensitive), a macOS notification is sent:

> **Cursor needs approval**
> Command approval needed

### Requirements

- macOS (uses `osascript` for notifications)
- `script` command (standard on macOS)

### How it works

1. Runs the given command under `script` to capture its output (preserving TTY behavior)
2. Tails the output file in the background
3. When a line matches the approval pattern, triggers `osascript` to display a notification
4. Cleans up the temp file on exit
