# flawless

**Ship branches through a quality gate, without the ceremony.**

flawless validates your branch in a disposable git worktree — your AI
agent reviews the diff, your tests and linters run, safe problems are
fixed automatically — then pushes the branch and opens the PR. One
static binary. No daemon, no hooks, no gate repository, no telemetry,
zero dependencies.

```text
$ flawless
flawless feature/search → origin/main  (agent: claude, run: 20260713-104200-feature-search)
✓ sync    rebased onto origin/main (1.2s)
▸ review  agent code review of the diff
    [blocker] search.go:88 query is interpolated into SQL unescaped
  1 blocking finding(s) in review — what now? [f]ix [a]ccept [s]kip step [q]uit f
✓ review  0 findings, none blocking (41s)
✓ test    go test ./... — passed (12s)
✓ lint    go vet ./... — passed (2s)
✓ push    pushed 8446086cc5 to origin/feature/search (0.8s)
✓ pr      https://github.com/you/repo/pull/42 (2.1s)
✦ flawless: all gates passed — https://github.com/you/repo/pull/42
```

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/3cpo-dev/flawless/main/install.sh | sh
```

or, with Go 1.22+:

```sh
go install github.com/3cpo-dev/flawless@latest
```

## Use

```sh
cd your-repo
git checkout -b my-change      # commit some work
flawless                       # that's it — no init required
```

Useful flags: `--intent "what this change should do"`, `--yes`
(non-interactive), `--detach` (background; follow with `flawless logs
-f`), `--skip review,lint`, `--json` (machine-readable events for
coding agents).

Check your environment with `flawless doctor`. Optionally write a fully
commented config with `flawless init` — most repos only ever set:

```yaml
# .flawless.yaml
commands:
  test: make test
  lint: make lint
```

## How it works

```text
flawless
  ├─ git worktree add --detach   isolated copy; your checkout is never touched
  ├─ sync                        fetch + rebase onto the target branch
  ├─ review                      agent reviews the diff against your intent; blockers gate
  ├─ test / lint                 your commands, with a bounded agent auto-fix loop
  ├─ docs (opt-in)               agent refreshes docs the diff made stale
  ├─ push                        the validated SHA, lease-protected
  ├─ pr                          created/updated via gh
  └─ ci (opt-in)                 watch checks via gh
```

Blocking findings always stop the run: interactively you choose
fix/accept/skip/quit; under `--yes` flawless fixes what is safely
fixable and fails on the rest. It never silently ships a blocker, and
every auto-fix is its own commit so the PR shows exactly what the
pipeline changed.

State is plain JSON in `.git/flawless/runs/` — `flawless status`,
`flawless runs` and `flawless logs` are thin readers over files you can
also just `cat`.

**Is the gate mandatory?** By default, no — `flawless` is a command, so
you *could* still `git push origin` around it. Solo, that freedom is a
feature. For teams, `flawless guard on` installs a pre-push hook that
refuses direct pushes to the gated remote (flawless's validated pushes
pass; `FLAWLESS_BYPASS=1 git push …` is the visible emergency exit).

## Design

flawless is a deliberate simplification of the daemon-based push-gate
design (as popularized by
[no-mistakes](https://github.com/kunchenguid/no-mistakes)):

| Daemon-based gates | flawless |
| --- | --- |
| launchd/systemd/Task Scheduler daemon, sockets, locks | no resident process; `--detach` for background runs |
| local bare gate repo + hook + extra remote | none — the trigger is the `flawless` command |
| SQLite state, JSON-RPC | JSON files in `.git/flawless/` |
| TUI + separate agent command surface | plain output; `--json` is the agent API |
| installer manages services; updates restart daemons | replace one binary |
| embedded telemetry | none |

**Docs:** https://3cpo-dev.github.io/flawless/ — start with the
[Quick Start](https://3cpo-dev.github.io/flawless/start-here/quick-start/).

**Agents:** works out of the box with Claude Code (`claude`) and Codex
(`codex`), with any CLI via a one-line custom command, or with
`--agent none` as a pure rebase-test-lint-push-PR gate. A ready-made
Claude Code skill lives in [`skills/flawless/`](skills/flawless/SKILL.md).

## Development

```sh
make build   # build ./flawless
make test    # go vet + go test ./...
make docs    # build the docs site (docs/)
```

MIT licensed. Contributions welcome — run `flawless` on your branch
before opening the PR, naturally.
