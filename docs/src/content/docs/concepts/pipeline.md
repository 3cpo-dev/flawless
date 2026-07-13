---
title: Pipeline
description: The eight steps a flawless run can execute, what each does, and how failures are handled.
---

A run executes up to eight steps in a fixed order. Two always run
(`sync`, `push`); the rest are on by default or opt-in:

| # | Step | Default | What it does |
| --- | --- | --- | --- |
| 1 | `sync` | always | fetch the remote, rebase the worktree onto the target branch |
| 2 | `review` | on | agent code review of the diff; blocking findings gate the run |
| 3 | `test` | on | run the test command, auto-fix failures (up to `auto_fix.test`) |
| 4 | `lint` | on | run the lint command, auto-fix failures (up to `auto_fix.lint`) |
| 5 | `docs` | **opt-in** | agent updates documentation the diff made stale |
| 6 | `push` | always | push the validated SHA to the remote branch |
| 7 | `pr` | on | create the PR (or note the existing one) via `gh` |
| 8 | `ci` | **opt-in** | watch CI checks on the pushed branch via `gh` |

Steps disabled in config simply don't appear in the output. Steps skipped
by circumstance (empty diff, no test command detected, `gh` missing) are
shown as skipped with the reason — a skip is always explained.

## Step outcomes

Every step ends in exactly one of three states:

- **ok** — possibly with detail like `passing after 2 auto-fix(es)`.
- **skipped** — with the reason (`no lint command configured or detected`).
- **failed** — the run stops, the reason is recorded in the run file, and
  the exit code is non-zero. Nothing has been pushed unless the `push`
  step itself already succeeded.

## Where the agent is involved

Only three steps ever invoke the agent, and each with a narrow contract:

- **review** — read-only. The agent receives the diff and your intent and
  must answer with structured findings (severity, file, line, fix hint,
  `auto_fixable`). Prose answers are treated as "no findings" — flawless
  never guesses blockers out of free text.
- **test / lint auto-fix** — write access *inside the worktree only*. The
  agent gets the failing command and its output, with standing orders:
  minimal edits, and never weaken a test to make it pass.
- **docs** (opt-in) — write access, documentation files only.

If no agent is installed (or `agent: none`), review and docs are
disabled and test/lint failures gate immediately — flawless degrades into
a fast, honest "rebase, test, lint, push, PR" tool.

## The intent

The `--intent` flag ("add rate limiting to the search API") travels
through the whole pipeline: the review judges the diff *against the
intent*, fix passes are told not to change it, and it becomes the PR
title. Without the flag, flawless falls back to the branch's commit
subjects.

## Interactive gates

When blocking findings survive the auto-fix budget, the run pauses:

```text
1 blocking finding(s) in review — what now? [f]ix [a]ccept [s]kip step [q]uit
```

- **fix** — one more agent fix round, then re-review.
- **accept / skip** — proceed; the acceptance is recorded in the run detail.
- **quit** — abort the run; nothing is pushed.

Under `--yes`, `--json`, `--detach`, or when stdin is not a terminal, the
policy is mechanical: auto-fix when the budget allows and every blocker
is marked `auto_fixable`, otherwise **fail**. See
[Auto-Fix Loop](/flawless/concepts/auto-fix/).
