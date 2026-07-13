---
title: Using flawless from an Agent
description: Let your coding agent push its own work through the gate with --json and --yes.
---

flawless is designed to be driven *by* a coding agent as easily as by a
human — without a separate "agent API". The contract is three flags and
an exit code.

## The contract

```sh
flawless --yes --json --intent "<what the change is supposed to do>"
```

- `--yes` — no interactive gates. Blockers that can't be safely
  auto-fixed fail the run.
- `--json` — every event becomes one JSON object per line on stdout:

```json
{"event":"run_start","message":"flawless feature → origin/main …","ts":"2026-07-13T09:42:00Z"}
{"event":"step_start","step":"review","detail":"agent code review of the diff","ts":"…"}
{"event":"finding","severity":"blocker","location":"search.go:88","issue":"SQL built by string concatenation","ts":"…"}
{"event":"step_end","step":"review","status":"ok","detail":"passing after 1 auto-fix(es)","seconds":41.2,"ts":"…"}
{"event":"error","message":"test: test failed (go test ./...)","ts":"…"}
```

- **Exit code** — `0`: pushed and PR open; non-zero: the run stopped,
  and the last `error` event (plus `flawless logs`) says why.

Event types: `run_start`, `step_start`, `step_end`, `finding`, `info`,
`warning`, `error`.

## A skill for Claude Code

The repository ships a ready-made skill in `skills/flawless/SKILL.md`.
Copy it into your project (`.claude/skills/flawless/`) and Claude Code
can finish any task with `/flawless` — committing its work, running the
gate, and reacting to findings by fixing and re-running rather than
pushing past them.

The short version, for any agent:

1. Commit your work on a feature branch.
2. Run `flawless --yes --json --intent "<the task you were given>"`.
3. If it exits non-zero, read the `error` and `finding` events, fix the
   code (in the main checkout), commit, and run it again.
4. Report the PR URL from the final `run_start`…`step_end: pr` event —
   or from `flawless status`.

## Why there is no daemon API

In the daemon-based ancestor of this design, agents needed a parallel
command surface (`axi run`, `axi respond`, `axi status`, `axi logs`) to
converse with a resident process. Because a flawless run is just a
process with structured stdout and an exit code, the "API" is the same
one humans use — nothing to park, poll or resume.
