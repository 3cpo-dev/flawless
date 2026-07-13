---
title: The Gate Model
description: How flawless gates a push with no daemon, no bare repo and no hooks.
---

A quality gate needs three things: a trigger, an isolated place to
validate, and a controlled way to publish. Traditional push-gate tools
build all three out of infrastructure — flawless builds them out of
things git already has.

## The traditional design (and its cost)

The design flawless descends from intercepts `git push` itself: an extra
named remote points at a local bare repository, whose `post-receive` hook
notifies a background daemon over a socket; the daemon owns worktrees,
state (SQLite) and concurrency, and a TUI attaches to it via JSON-RPC.

It works, but every piece is a liability: services to register per
platform, a daemon that must be up before a push means anything, lock
files, crash recovery, "daemon refuses to restart during active runs",
and an update path that has to reset all of it.

## The flawless design

flawless observes that **the trigger doesn't need to be `git push`**.
You were going to type a command anyway — let it be `flawless` instead of
`git push origin branch`. Once the trigger is a foreground command, the
whole tower disappears:

```text
flawless
  │
  ├─ 1. reads your branch's HEAD           (nothing is sent anywhere)
  ├─ 2. git worktree add --detach /tmp/…   (isolated copy, your checkout untouched)
  ├─ 3. runs the pipeline in the worktree  (rebase, review, test, lint, fixes)
  ├─ 4. git push origin <validated-sha>    (the ONLY publishing moment)
  ├─ 5. gh pr create                       (or updates the existing PR)
  └─ 6. git worktree remove                (nothing left behind)
```

- **Trigger** — you, running a command. No hook, no daemon to be alive.
- **Isolation** — a disposable `git worktree` sharing the object store
  with your repo: creation is instant and costs almost no disk.
- **Publishing** — an ordinary `git push` of a specific validated SHA,
  guarded by `--force-with-lease` only when the rebase rewrote history,
  so a teammate's concurrent push is never clobbered.

## Trust properties

The gate is only worth having if it can't be quietly bypassed *by the
pipeline itself*:

1. **Nothing reaches the remote before the push step.** Review, test and
   lint all operate on a local worktree.
2. **Blocking findings stop the run** — interactively they need your
   explicit `fix`/`accept`/`skip`; under `--yes` anything the agent can't
   safely fix fails the run. There is no configuration that ships a
   blocker silently.
3. **Your local branch is never rewritten.** After a successful push,
   flawless fast-forwards your branch only when that is a pure
   fast-forward on a clean tree; otherwise it prints the `git pull
   --rebase` command and leaves the decision to you.
4. **A teammate's commits can never be overwritten.** At `sync`, the run
   pins the remote branch's exact SHA and verifies your local work
   already contains it (if not, the run fails with "pull first"). The
   final push demands the remote *still* be at that pinned SHA — if
   anyone pushed during the run, git refuses the push and flawless tells
   you to fetch, rebase and re-run.

## Enforcement: voluntary by default, structural on request

Be clear-eyed about one trade-off: because the trigger is a command, not
a hijacked `git push`, nothing physically stops you from running
`git push origin` and skipping the gate. For a solo developer that
freedom is a feature. For a team that wants the gate to be *the* path:

```sh
flawless guard on
```

This installs a `pre-push` hook that refuses direct pushes to the gated
remote. Flawless's own validated pushes pass through; everything else is
told to use the gate — with an explicit escape hatch for emergencies:

```sh
FLAWLESS_BYPASS=1 git push origin my-branch   # deliberate, visible, greppable
flawless guard off                            # remove the hook entirely
```

The guard is one shell script in `.git/hooks/pre-push` (it refuses to
touch a pre-push hook it didn't write). Still no daemon — the
enforcement is a file, and you can read it.

## What about "non-blocking pushes"?

The daemon design exists so `git push` returns instantly while validation
continues. flawless gives you the same property with one flag and zero
infrastructure:

```sh
flawless --detach     # returns immediately, run continues in the background
flawless logs -f      # attach to the log stream whenever you like
```

A detached run is a normal OS process. If it dies, `flawless status`
says `crashed` — determined by checking whether the recorded PID is
alive, not by a recovery daemon.
