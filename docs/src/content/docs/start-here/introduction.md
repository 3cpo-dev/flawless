---
title: Introduction
description: What flawless is, why it exists, and how it differs from heavier push-gate tools.
---

**flawless** is a pre-push quality gate. Before your branch reaches the
remote, it is validated in an isolated git worktree: your AI coding agent
reviews the diff, your tests and linters run, safe problems are fixed
automatically, and only then is the branch pushed and a pull request
opened.

The pitch in one line: *kill the slop before it ships — with one binary
and no ceremony.*

## The problem

AI-assisted development produces a lot of code fast, and some of it is
subtly wrong: tests that were never run, review comments nobody asked
for, half-updated docs, lint noise. Teams end up re-reviewing everything
by hand, which throws away the speed they just gained.

A quality gate between "committed" and "shared" fixes this — but existing
gate tools bring a lot of machinery: background daemons registered with
launchd/systemd, local bare "gate" repositories, git hooks, IPC sockets,
SQLite state, TUIs, setup wizards. That machinery is itself a source of
failures, update pain and trust issues.

## The flawless answer

flawless keeps the gate and deletes the machinery:

| Heavier gate tools | flawless |
| --- | --- |
| Background daemon (launchd/systemd/Task Scheduler) | No daemon. A run is a normal process; add `--detach` to background it |
| Local bare gate repo + `post-receive` hook + extra remote | No gate repo. `flawless` works directly from your branch |
| SQLite database, JSON-RPC over sockets | Plain JSON files in `.git/flawless/` — the file *is* the API |
| TUI + separate agent interface | Plain terminal output; `--json` turns every line into a machine-readable event |
| Setup wizard, `init`, `eject`, daemon restart | Nothing to install into the repo; `flawless init` only writes an optional commented config file |
| Embedded telemetry | None |

The result is one static binary you can read end-to-end, that starts
working the moment it lands on your `PATH`.

## What a run looks like

```text
$ flawless
flawless feature/search → origin/main  (agent: claude, run: 20260713-104200-feature-search)
✓ sync    rebased onto origin/main (1.2s)
▸ review  agent code review of the diff
    [blocker] search.go:88 query is interpolated into SQL unescaped
  1 blocking finding(s) in review — what now? [f]ix [a]ccept [s]kip step [q]uit f
✓ review  0 findings, none blocking (41s)
✓ test    go test ./... — passing after 0 auto-fix(es) (12s)
✓ lint    go vet ./... — passed (2s)
✓ push    pushed 8446086cc5 to origin/feature/search (0.8s)
✓ pr      https://github.com/you/repo/pull/42 (2.1s)
✦ flawless: all gates passed — https://github.com/you/repo/pull/42
```

## Design principles

1. **The human rules.** Blocking findings pause the run for your decision:
   fix, accept, skip or quit. In non-interactive mode (`--yes`), flawless
   fixes what is safely fixable and *fails* on anything else — it never
   silently ships a blocker.
2. **Your checkout is sacred.** All validation, fixing and rebasing
   happens in a disposable worktree. The only writes to your working copy
   are an optional fast-forward after a successful push.
3. **No resident software.** Nothing runs when flawless isn't running.
   There is no state that can drift, no service to restart, no lock to
   force.
4. **Boring interfaces.** Files, exit codes and JSON lines. Everything
   flawless knows is inspectable with `cat` and `ls`.

## Next steps

- [Quick Start](/flawless/start-here/quick-start/) — first gated push in two minutes.
- [Installation](/flawless/start-here/installation/) — all install methods.
- [The Gate Model](/flawless/concepts/gate-model/) — how the gate works without a daemon.
