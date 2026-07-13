---
title: CLI Commands
description: Every flawless command and flag.
---

## `flawless` (the run command)

Validates the current branch in a disposable worktree, pushes it and
opens a PR. This is the default command — no subcommand needed.

| Flag | Meaning |
| --- | --- |
| `--intent <text>` | what this branch is meant to do; guides the review and titles the PR. Default: inferred from commit subjects |
| `--yes` | non-interactive: auto-fix what is safe, fail on anything blocking that isn't |
| `--skip <steps>` | comma-separated steps to skip this run: `sync,review,test,lint,docs,push,pr,ci` |
| `--detach` | run in the background (Unix only); follow with `flawless logs -f` |
| `--json` | emit machine-readable JSON events on stdout (implies non-interactive gates) |
| `--target <branch>` | base branch to rebase onto / open the PR against. Default: the remote's default branch |
| `--remote <name>` | git remote to push to. Default: `origin` |
| `--agent <spec>` | agent override: `auto`, `claude`, `codex`, `none`, or a custom command containing `{prompt_file}` |

Exit code `0` means every executed step passed and the branch is pushed.
Non-zero means the run stopped; `flawless logs` has the detail.

## `flawless init`

Writes a fully commented `.flawless.yaml` in which every value is the
default. Refuses to overwrite an existing file. flawless does **not**
require init — this is documentation you can uncomment.

## `flawless guard`

Makes the gate mandatory for this repo via a `pre-push` hook.

| Subcommand | Meaning |
| --- | --- |
| `on` | install the hook: direct `git push` to the gated remote is refused (flawless's own pushes pass; `FLAWLESS_BYPASS=1 git push …` bypasses once) |
| `off` | remove the hook |
| `status` (default) | report whether the guard is installed |

The hook is a plain shell script; `guard on`/`off` refuse to overwrite
or delete a pre-push hook flawless does not own.

## `flawless status`

Shows the latest run: branch, status, PR URL, and each step's outcome
with detail. A run recorded as `running` whose process no longer exists
is reported as `crashed`.

| Flag | Meaning |
| --- | --- |
| `--all` | list run history instead (newest first) |
| `--limit <n>` | max rows with `--all` (default 20) |

`flawless runs` is an alias for `flawless status --all`.

## `flawless logs`

Prints the latest run's log.

| Flag | Meaning |
| --- | --- |
| `-f` | follow while the run is active (also prints the final status) |
| `--full` | whole log instead of the tail |
| `-n <lines>` | tail length (default 40) |

## `flawless doctor`

Checks git, the repository, config validity, agent availability, `gh`,
the remote and the target branch. Exit code is non-zero when any check
fails, so it can be used in setup scripts.

## `flawless version`

Prints the version stamped at build time (`dev` for source builds).

## What's deliberately absent

No `daemon start/stop/restart`, no `attach`, no `eject`, no `axi *`, no
`update`, no `stats`. Each absence is a feature: there is no resident
process to manage, nothing installed into your repo to remove, the
[agent interface](/flawless/guides/agent-mode/) is the ordinary CLI, updating
is replacing one binary, and flawless does not phone home to count your
usage.
