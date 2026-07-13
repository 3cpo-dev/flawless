---
name: flawless
description: Push the current branch through the flawless quality gate (agent review, tests, lint), then open the PR. Use when the user asks to ship, push, or gate their work, or says /flawless.
---

# Ship the current branch through flawless

flawless validates committed work in an isolated worktree, pushes the
branch and opens a PR. Exit code 0 = shipped; non-zero = the gate
stopped the run and the events say why.

## Steps

1. Ensure the work is committed on a feature branch (not the default
   branch). Commit pending changes that belong to the task; leave
   unrelated changes uncommitted — flawless validates commits only.
2. Run:

   ```sh
   flawless --yes --json --intent "<one sentence: what this change is supposed to do>"
   ```

   Use the user's task description as the intent — the review judges
   the diff against it, so make it accurate, not flattering.
3. If the exit code is non-zero, read the `finding` and `error` JSON
   events (and `flawless logs` for full output). Fix the underlying
   problem in the main checkout, commit, and run the same command
   again. Do not bypass findings with `--skip` unless the user
   explicitly tells you to.
4. On success, report the PR URL (in the `pr` step_end event, or from
   `flawless status`) and summarize any auto-fix commits the pipeline
   added (`git log` after `git pull --rebase` if flawless said the
   local branch was not fast-forwarded).

## Notes

- `flawless doctor` diagnoses a missing agent/remote/gh setup.
- A rebase conflict means: rebase manually onto the target branch,
  resolve, then re-run flawless.
- Never run flawless with `--detach` from this skill; you need the exit
  code.
