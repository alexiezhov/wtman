---
name: wtman
description: Drives the `wtman` multi-repo git worktree manager — creating, listing, updating, renaming, and deleting feature branches across many repos at once, plus initial install/config setup. Use this skill whenever the user mentions wtman, asks to "spin up a feature branch across repos", wants to manage git worktrees over multiple repositories, needs the multi-root Cursor/VS Code workspace generated, or needs help installing/configuring wtman (config file, source_dir, target_dir, post_command). Also use it when the user issues short commands like "wtman ls", "new branch foo in repos x,y", "delete the worktree for bar", or any feature-branch lifecycle action that wtman owns.
---

# wtman

`wtman` is a multi-repo git worktree manager. It creates feature branches across many repositories at once under a shared directory, generates a Cursor/VS Code multi-root workspace, and exposes a CLI for scripting and a TUI for interactive use.

This skill operates in one of two modes. Decide which **before** doing anything else.

## Picking the mode

Run this single probe first:

```bash
command -v wtman && cat ~/.config/wtman/config.json 2>/dev/null
```

- **Setup mode** if any of these are true:
  - `wtman` is not on `PATH`,
  - `~/.config/wtman/config.json` is missing,
  - the config's `source_dir` or `target_dir` is empty or points at a directory that doesn't exist,
  - the user explicitly asked to "install", "set up", "configure", or "initialize" wtman.
- **Normal mode** otherwise.

If you land in setup mode (including when the user *asked* for a normal command but the install/config is incomplete), tell them why and **read `references/setup.md`** for the full install + config workflow. That file is the source of truth for installation and setup; the rest of this document covers normal operation only.

---

## Normal mode

Translate the user's request into the right `wtman` invocation and report the result. All commands emit JSON on stdout — parse it and render a short human summary, don't dump raw JSON unless the user asks.

### Command map

| User intent | Command |
|---|---|
| List feature branches / "what worktrees do I have?" | `wtman ls` |
| List discoverable source repos | `wtman repos` |
| Create a new branch across repos | `wtman new <branch> <repo1,repo2,...> [--from <ref>] [-n]` |
| Add/remove repos on an existing branch | `wtman update <branch> <repo1,repo2,...>` |
| Rename a feature branch directory | `wtman mv <old> <new>` |
| Pull every source repo | `wtman pull` |
| Delete a feature branch (worktrees + dir) | `wtman rm <branch>` |
| Check the installed version | `wtman version` |

Flag notes:
- `-n` on `wtman new` skips the configured `post_command` (useful when the user just wants the worktrees and will open them themselves).
- `--from <ref>` bases new branches off something other than `main`/`master` (a branch, tag, or commit). wtman resolves it per-repo as a local ref first, then `origin/<ref>`.
- Flags can appear before, after, or between positional args.

### Resolving repo names

Repo names in `wtman new` / `update` must match what `wtman repos` returns. Before constructing a command:

- If the user names specific repos, validate them against `wtman repos`. If a name is missing or ambiguous (typo, partial match), surface the available list and ask which they meant — don't guess.
- If the user says "all repos" or doesn't specify, run `wtman repos` and pass every name. Confirm with the user before creating worktrees in *every* repo, since that can be a lot of disk and a slow `post_command`.

### Safety — destructive operations

`wtman rm` and `wtman update` can lose work if a worktree has uncommitted changes. wtman's own safety model:

- Without `-f`, both commands refuse to delete a dirty worktree. `update` returns `{"error":"dirty","repos":[...]}` and exits non-zero; `rm` errors out.
- With `-f`, they nuke dirty worktrees with no further prompts.

Your rules:

1. **Never pass `-f` on the first try.** Always attempt the safe version first.
2. If wtman reports the operation is blocked because something is dirty:
   - Tell the user *which* repos are dirty and what's in them. For each dirty repo, run `git -C <branchDir>/<repo> status --short` and show the output.
   - Offer concrete alternatives in this order, and let the user pick:
     a. Stash or commit the changes inside the dirty worktree(s), then retry.
     b. Keep the dirty worktree, drop only the clean ones (use `wtman update <branch> <clean-repos>`).
     c. Force-delete with `-f`, losing the uncommitted changes.
   - Only run with `-f` after the user explicitly confirms option (c). Quote back what will be lost ("I'll force-delete `repo-a` and `repo-b`, discarding their uncommitted changes — confirm?") and wait for a clear yes.
3. For `wtman mv`, double-check the new name doesn't already exist (run `wtman ls` first). Renaming over an existing branch silently is bad.
4. For `wtman pull`, warn that it touches every repo under `source_dir` and may take a while; otherwise it's read-mostly and safe.

### Reading state before acting

When the user's request is vague ("clean up that old branch", "update my refactor worktree"), run `wtman ls` first and confirm which branch they mean by name before doing anything destructive. Branches are matched exactly — there's no fuzzy match in wtman.

### Reporting results

After a successful command, give the user a one-line summary plus the path(s) they care about. Examples:

- After `new`: "Created `feat-x` in 3 repos at `/Users/.../branches/feat-x`. Workspace: `.../feat-x.code-workspace`."
- After `ls` with several branches: render a short table (name, date, repos) — not the raw JSON.
- After `rm`: "Removed `feat-x` and its 3 worktrees."

If wtman exits non-zero, surface its stderr verbatim — its error messages are short and specific, and rewriting them loses information.

### Debugging

If something looks off and you want more detail, rerun with `-v` or `--log-level debug`. Logs go to stderr; stdout stays clean JSON.

---

## What this skill does *not* do

- Doesn't manage what's inside the worktrees (commits, PRs, rebases). That's the user's job with plain `git`.
- Doesn't edit the `colors` block in config — that's a TUI concern.
- Doesn't install or configure the Cursor approval watcher (`watcher.sh`) unless the user explicitly asks. See `README.md` in the wtman repo.

## Files in this skill

- `references/setup.md` — Install + config workflow. Read **only** in setup mode.
