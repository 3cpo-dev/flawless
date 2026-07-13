---
title: Choosing an Agent
description: How flawless selects and drives an AI agent CLI, and how to plug in your own.
---

flawless bundles no model and stores no API keys. It drives the agent
CLI you already use, as a subprocess, inside the disposable worktree.

## Selection

```yaml
agent: auto
```

- `auto` (default) — first available of: `claude`, `codex`.
- `claude` / `codex` — force one; `flawless doctor` verifies it exists.
- `none` — disable agent involvement entirely. Review and docs steps
  turn off; test/lint failures gate immediately instead of auto-fixing.
  flawless remains a fast "rebase, test, lint, push, PR" gate.
- anything else — a **custom agent command** (below).

Override per run with `--agent`:

```sh
flawless --agent none          # infra-only gate, no AI today
flawless --agent codex
```

## How the built-in agents are invoked

| Agent | Read-only passes (review) | Write passes (fixes, docs) |
| --- | --- | --- |
| `claude` | `claude -p <prompt> --output-format text` | adds `--permission-mode acceptEdits` |
| `codex` | `codex exec <prompt>` | adds `--sandbox workspace-write` |

Write access is only ever granted inside the worktree, and each call has
a 15-minute timeout.

## Custom agent command

Any CLI that can read a prompt and print an answer works. Set `agent` to
a shell command containing `{prompt_file}`:

```yaml
agent: "aider --message-file {prompt_file} --yes"
# or
agent: "ollama run qwen3 < {prompt_file}"
```

The command runs with the worktree as its working directory. For the
review pass, its stdout must contain a JSON object in this shape
(anywhere in the output — code fences and surrounding prose are fine):

```json
{"findings": [
  {"severity": "blocker", "file": "a.go", "line": 12,
   "issue": "what is wrong", "fix": "how to fix it", "auto_fixable": true}
]}
```

`severity` is `blocker`, `warning` or `info`; only blockers gate the
run. Output with no parsable JSON is treated as "no findings" (with a
warning), never as a blocker.

For fix and docs passes, the command is expected to edit files in place;
flawless commits whatever changed.

## Practical guidance

- **Give the review something to judge against.** `--intent "…"` turns
  the review from "is this good code?" into "does this do what the
  author meant?" — a much higher-signal question.
- **Keep fix budgets small.** The default (3 for test/lint, 1 for
  review) is deliberate; see [Auto-Fix Loop](/flawless/concepts/auto-fix/).
- **Agent flaky today?** `--skip review` gates on tests and lint only,
  without touching your config.
