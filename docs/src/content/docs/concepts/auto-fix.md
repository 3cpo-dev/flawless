---
title: Auto-Fix Loop
description: How flawless decides what the agent may fix automatically, and what always needs a human.
---

flawless splits problems into two classes and refuses to blur the line:

- **Mechanical problems** — a failing test whose cause is obvious, lint
  errors, a review finding marked `auto_fixable`. The agent may fix these
  automatically, within a per-step budget.
- **Judgment problems** — anything touching intent, design, or a finding
  the reviewer did not mark safely fixable. These pause the run for you,
  or fail it in non-interactive mode. **There is no setting that ships a
  judgment problem silently.**

## The loop

For `test` and `lint`:

```text
run command ──fail──▶ agent fix (budget left?) ──▶ commit ──▶ run command again
     │                        │
     ok                       no budget ──▶ gate: [f]ix [a]ccept [s]kip [q]uit
     ▼                                        (--yes: fail)
   step ok
```

For `review`, the loop re-reviews the *new* diff after each fix round, so
a fix that introduces a new problem is caught by the same gate that
requested it.

## Budgets

```yaml
auto_fix:
  review: 1   # fix rounds for blocking review findings
  test: 3     # fix attempts for a failing test command
  lint: 3
```

Budgets are deliberately small. If the agent hasn't fixed a test in three
attempts, the fourth attempt is rarely the one that works — a human
should look. Setting a budget to `0` disables automation for that step
entirely: the gate then only offers accept/skip/quit.

## Every fix is a commit

Each successful fix round becomes its own commit in the worktree
(`fix: test findings (flawless auto-fix)`), so the PR shows exactly what
the pipeline changed, separately from what you wrote. Nothing is ever
amended into your commits.

## Guardrails baked into the prompts

The fix prompts carry standing orders the agent sees on every invocation:

- minimal, targeted edits — no drive-by refactoring;
- never delete or weaken a test to make it pass, unless the test is
  provably wrong and the reason is stated in a comment;
- never run git commands — flawless owns the repository state;
- the author's intent is quoted and marked do-not-change.

Guardrails are not guarantees — that's why the budget exists, why every
fix is re-validated by the same command or review that flagged it, and
why the whole thing happens in a worktree you can inspect before anything
is pushed.
