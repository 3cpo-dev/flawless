---
title: Pipeline Steps
description: Exact behavior of each step — inputs, outcomes, and failure semantics.
---

Steps run in this order. "Gate" means blocking findings pause an
interactive run or fail a non-interactive one.

## sync — always runs

Fetches the remote, pins where `<remote>/<branch>` currently is (the
push step will hold it to exactly that SHA), then rebases the worktree's
HEAD onto `<remote>/<target>`.

- The remote branch has commits your local branch doesn't include →
  **failed** with "pull first" — pushing would have discarded them.
- Target branch missing on the remote → **skipped** (first push of a new
  lineage).
- Conflicts → rebase aborted, **failed** with instructions; your
  checkout is untouched.

## review — default on, needs an agent

Sends `git diff <remote>/<target>...HEAD` (minus `ignore` prefixes) and
the intent to the agent, read-only. Expects structured findings;
unparsable output = no findings (warned, never invented into a blocker).

- Findings printed with severity and location; **blockers gate**.
- `fix` at the gate (or auto-fix under budget when every blocker is
  `auto_fixable`) → agent edits, flawless commits, the **new** diff is
  re-reviewed.
- Empty diff → **skipped**.

## test / lint — default on

Runs `commands.test` / `commands.lint` (`sh -c`, in the worktree), or
the auto-detected equivalent; skipped with reason when neither exists.

On failure: agent fix → commit → re-run, up to `auto_fix.<step>` times;
then the gate. Accepting a failure at the gate is recorded in the run
detail (`test failure accepted by user`).

## docs — opt-in (`steps.docs: true`), needs an agent

Agent updates documentation the diff made stale — documentation only,
committed as its own commit. No stale docs → step reports
`documentation already adequate`.

## push — always runs

Pushes the validated worktree SHA to `<remote>/<branch>`.

- Remote already at that SHA → **skipped**.
- The push carries `--force-with-lease=<branch>:<sha pinned at sync>`
  (empty = branch must not exist yet). If anyone pushed to the branch
  at any point the run didn't incorporate — even before the run's own
  fetch — git refuses and flawless reports that nothing was
  overwritten.
- Afterwards, your local branch is fast-forwarded only if it's checked
  out on a clean tree and the move is a pure fast-forward; otherwise
  flawless prints the `git pull --rebase` to run.

## pr — default on, needs `gh`

Creates the PR (`--head <branch> --base <pr.base|target>`, title =
intent or first commit subject, body lists the gates that passed;
`pr.draft: true` for drafts). An existing PR is left in place — the push
already updated it. `gh` missing → **skipped**, never failed.

## ci — opt-in (`steps.ci: true`), needs `gh`

Watches the PR's checks (`gh pr checks --watch`). Green → ok; a failing
check → **failed** with the check output. flawless does not auto-fix CI
in v1 — by this point the code passed review, tests and lint locally,
so a CI failure usually means environment drift worth human eyes.
