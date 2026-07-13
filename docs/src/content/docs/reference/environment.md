---
title: Environment Variables
description: The complete list — it is short on purpose.
---

flawless reads exactly three environment variables:

| Variable | Effect |
| --- | --- |
| `NO_COLOR` | any value disables colored output ([no-color.org](https://no-color.org)) |
| `HOME` | locates the global config, `~/.config/flawless/config.yaml` |
| `TMPDIR` | where disposable worktrees are created (`$TMPDIR/flawless-<run-id>`) |

One variable is set *by* flawless, for itself:

| Variable | Effect |
| --- | --- |
| `FLAWLESS_DETACHED=1` | marks the re-executed background process of a `--detach` run: output goes to the run log and gates resolve non-interactively. Not intended to be set by hand |

Everything else that could have been an environment variable is either a
flag (`--target`, `--agent`), a config field, or doesn't exist because
the feature it would configure (daemon home, socket paths, telemetry
opt-out, update channels) doesn't exist.

Agent subprocesses inherit your environment unchanged, so whatever
authentication your `claude`/`codex`/`gh` already use keeps working —
flawless adds no environment of its own.
