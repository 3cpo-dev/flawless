---
title: Troubleshooting
description: The problems you might actually hit, and what to do about each.
---

First move, always:

```sh
flawless doctor     # environment and repo checks
flawless logs       # everything the last run printed
flawless status     # per-step outcomes of the last run
```

## "no agent CLI found"

flawless looked for `claude` and `codex` on `PATH`. Either install one,
set a [custom agent command](/flawless/guides/agents/), or run without
AI: `flawless --agent none`.

## "rebase onto origin/main hit conflicts"

Your branch and the target diverged in the same lines. flawless aborts
the rebase in the worktree (your checkout was never involved) and stops.
Rebase manually in your checkout, resolve, then re-run:

```sh
git fetch origin && git rebase origin/main
flawless
```

## "you are on "main", the target branch itself"

flawless gates feature branches. Create one:
`git checkout -b my-change` — then re-run.

## Test or lint step is skipped

Detail says `no test command configured or detected`. Detection covers
`Makefile` targets, `go.mod`, `package.json` scripts, `Cargo.toml` and
`pyproject.toml`. Anything else needs one line of config:

```yaml
commands:
  test: ./scripts/test.sh
```

## The push was rejected (`force-with-lease`)

Someone pushed to your branch after your run fetched. This is the
data-loss protection working. `git fetch origin`, look at what landed,
rebase onto it, re-run.

## `pr` / `ci` steps are skipped

`gh` is not installed or not authenticated (`gh auth login`), or the
remote isn't GitHub. Everything up to and including `push` still ran —
these steps never fail a run by their absence.

## A detached run shows `crashed`

The background process died — machine slept hard, OOM, or a bug. The log
ends where it ended: `flawless logs`. Re-running is always safe; a run
is idempotent up to its push step, and the push is lease-protected.

## The review returned nothing / prose

flawless prints `agent review returned no parsable findings; treating as
clean` and moves on — an unparsable review is never invented into a
blocker. If it keeps happening with a custom agent, make sure it prints
the JSON shape from [Choosing an Agent](/flawless/guides/agents/).

## My local branch is "behind" after a run

The pipeline added fix commits and history allowed only a lease-guarded
push, so flawless didn't touch your checkout. It printed the exact
command; it's always:

```sh
git pull --rebase origin <your-branch>
```

## Where is everything?

| What | Where |
| --- | --- |
| run records + logs | `.git/flawless/runs/` |
| repo config | `.flawless.yaml` |
| global config | `~/.config/flawless/config.yaml` |
| worktrees (during a run only) | `$TMPDIR/flawless-<run-id>` |

Deleting any of these is safe; flawless recreates what it needs.
