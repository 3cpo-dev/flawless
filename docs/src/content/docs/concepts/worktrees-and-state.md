---
title: Worktrees & State
description: Where runs execute, where their state lives, and why there is no daemon to manage any of it.
---

## The worktree

Every run executes in a disposable worktree:

```sh
git worktree add --detach /tmp/flawless-<run-id> <your-HEAD>
```

Worktrees share the object database with your repository, so creating one
is instant and nearly free. Inside it, the pipeline can rebase, let the
agent edit files, and commit fixes — while your checkout, your index and
your uncommitted changes remain byte-for-byte untouched. When the run
ends (pass, fail or Ctrl-C), the worktree is removed.

Two consequences worth knowing:

- **Only committed work is validated.** If your tree is dirty, flawless
  warns you that uncommitted changes are not part of the run.
- **Agent blast radius is capped.** Even a badly misbehaving fix pass can
  only damage a directory that is about to be deleted.

## State: files, not a database

Everything flawless remembers lives under `.git/flawless/runs/`:

```text
.git/flawless/runs/
├── 20260713-104200-feature.json   # the run record
└── 20260713-104200-feature.log    # everything the run printed
```

The JSON record holds branch, target, intent, per-step outcomes with
durations, the validated SHA, the PR URL and the run's PID. `flawless
status`, `flawless runs` and `flawless logs` are thin readers over these
files — and so is `cat`, which is the point. There is no database to
corrupt, no schema to migrate, and the history is per-repo, so deleting
the repo deletes everything.

## Background runs without a daemon

`flawless --detach` starts the same process detached from your terminal
(its own session, output to the run log) and returns. That is the entire
"async architecture":

```sh
flawless --detach
flawless logs -f       # follow the log
flawless status        # passed / failed / running / crashed
```

Liveness is determined by checking the PID recorded in the run file. A
run whose process disappeared shows as `crashed` — honestly, and without
a recovery daemon that itself can crash.

## Concurrency

Runs are independent processes, so two branches can run simultaneously
from two terminals. Pushing the *same* branch twice concurrently is not
serialized by flawless — the second push's `--force-with-lease` will
refuse if it would lose the first one's commits, which is git's own,
well-tested arbitration. In practice: one branch, one run at a time, and
`flawless status` before re-running.
