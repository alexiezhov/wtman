# wtman setup — install and configure

Read this file only when you're in **setup mode** (the user asked to install/configure wtman, or you detected that the binary is missing or the config is incomplete). In normal mode this file is irrelevant and the main `SKILL.md` is enough.

The goal of setup: after you're done, `wtman repos` runs without error and lists the user's actual repositories.

---

## Step 0 — Probe current state

Run these in parallel before talking to the user, so you know exactly what's missing:

```bash
command -v wtman
command -v brew
command -v git
sw_vers -productName 2>/dev/null   # confirms macOS (wtman targets macOS)
cat ~/.config/wtman/config.json 2>/dev/null
```

Decide:

- If `wtman` resolves → skip to **Step 2** (config).
- If `wtman` is missing → start at **Step 1** (install).

If `sw_vers` shows something other than `macOS`, warn the user: wtman's `post_command` defaults and the watcher rely on macOS-only tools (`open`, `osascript`, `script`). Core worktree operations still work on Linux, but the user should know.

---

## Step 1 — Install the binary

### Prerequisites

- **Homebrew** — `brew --version` to check. If missing, point the user at [brew.sh](https://brew.sh); don't install it for them.
- **git** — `git --version` to check. Should always be present on a developer machine; if not, `brew install git`.

### Install via Homebrew

```bash
brew install --cask alexiezhov/apps/wtman
```

This taps `alexiezhov/homebrew-apps` and installs the latest release onto `PATH`; later upgrades are `brew upgrade --cask wtman`. Homebrew handles the install location and `PATH` for you.

### Verify

```bash
command -v wtman && wtman version
```

`wtman version` prints the version, commit, and build date. If a normal command later prints "config not found" or similar, that's expected before step 2.

### Optional — Cursor approval watcher

Only mention this if the user asks about Cursor agent mode (Cursor CLI), approval prompts, or `watcher.sh`. The watcher is independent of wtman itself:

```bash
cp watcher.sh ~/.local/bin/cursor-watcher
chmod +x ~/.local/bin/cursor-watcher
```

Don't proactively install it during normal setup — it's a separate tool.

---

## Step 2 — Write the config

Config path: `~/.config/wtman/config.json`. The full schema (matches `core/config.go` exactly):

```json
{
  "source_dir": "",
  "target_dir": "",
  "post_command": "open {{workspace}}",
  "scan_depth": 1,
  "log_level": "info"
}
```

Field meanings:

- `source_dir` — directory containing the user's git repositories. wtman scans here (`scan_depth` levels deep) to find repos.
- `target_dir` — directory where wtman creates one subdirectory per feature branch, each containing one worktree per included repo. Must differ from `source_dir`.
- `post_command` — shell command run after a successful `wtman new`. `{{dir}}` → branch directory; `{{workspace}}` → absolute path to the generated `.code-workspace` file.
- `scan_depth` — how deep wtman looks under `source_dir` for `.git` directories. `1` = direct children.
- `log_level` — `debug`, `info`, `warn`, `error`, or `off`. Logs go to stderr.

### Read what's already there

If the config file exists, load it and figure out which fields are missing or invalid:

- `source_dir` empty or non-existent on disk → ask.
- `target_dir` empty, equal to `source_dir`, or pointing somewhere weird → ask.
- `post_command` missing → use the default below.

Only prompt for fields that need fixing. Don't re-ask things that already look right.

### Prompt for `source_dir`

Ask: "Where do your git repositories live? (Absolute path to the parent directory wtman should scan.)"

Validate the answer:

```bash
test -d "<source_dir>"
find "<source_dir>" -maxdepth 2 -type d -name .git 2>/dev/null | head
```

If the `find` returns nothing, tell the user no `.git` directories were found at depth 1 and ask whether to:
- Change the path,
- Increase `scan_depth` (then re-check at that depth),
- Or proceed anyway (rare — usually a sign of a typo).

### Prompt for `target_dir`

Ask: "Where should wtman create the feature-branch worktrees? (A separate directory from `source_dir`.)"

Validate:

```bash
test -d "<target_dir>"
```

If it doesn't exist, ask before creating: "Create `<target_dir>`? (y/n)". On yes:

```bash
mkdir -p "<target_dir>"
```

Reject answers equal to `source_dir` or nested inside it.

### Pick `post_command`

Default to `"open {{workspace}}"` — this opens the generated multi-root workspace in whichever app handles `.code-workspace` on macOS (Cursor or VS Code). Confirm with the user and offer common alternatives:

- `"open -a Cursor {{workspace}}"` — force-open in Cursor (use if VS Code is the default `.code-workspace` handler but the user prefers Cursor).
- `"open -a 'Visual Studio Code' {{workspace}}"` — force-open in VS Code.
- `"tmux split-window -h 'cd {{dir}} && cursor --agent'"` — opens a tmux pane running Cursor agent CLI.
- `""` (empty) — do nothing after creation; the user opens it themselves.

Remind the user that `{{dir}}` and `{{workspace}}` are the only placeholders.

### `scan_depth` and `log_level`

Don't prompt for these unless something specific motivated it (e.g. step 2 above had to bump scan_depth, or the user mentioned wanting verbose logs). Use defaults: `1` and `"info"`.

### Show, confirm, write

Show the assembled config back to the user *before* writing:

```
About to write ~/.config/wtman/config.json:

{
  "source_dir": "/Users/me/dev",
  "target_dir": "/Users/me/branches",
  "post_command": "open {{workspace}}",
  "scan_depth": 1,
  "log_level": "info"
}

OK to write?
```

On confirmation, write via heredoc (never `echo` with shell-interpreted strings — paths may contain spaces or special chars):

```bash
mkdir -p ~/.config/wtman
cat > ~/.config/wtman/config.json <<'JSON'
{
  "source_dir": "/Users/me/dev",
  ...
}
JSON
```

---

## Step 3 — Verify and hand off

```bash
wtman repos
```

Render the discovered repos as a short list. If the output is empty or clearly wrong (missing repos the user mentioned), loop back to step 2 and revisit `source_dir` / `scan_depth`.

Once it's right, tell the user setup is complete and ask what they'd like to do next. Drop back into normal mode for that request — don't re-read this file.
