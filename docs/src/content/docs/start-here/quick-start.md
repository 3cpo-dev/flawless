---
title: Quick Start
description: From install to your first validated push in two minutes.
---

## 1. Install

```sh
curl -fsSL https://raw.githubusercontent.com/3cpo-dev/flawless/main/install.sh | sh
```

Other methods (Homebrew-style manual install, `go install`, from source)
are covered in [Installation](/flawless/start-here/installation/).

## 2. Check your environment

```sh
flawless doctor
```

```text
✓ git            git version 2.44.0
✓ repository     /Users/you/code/project
✓ config         valid
✓ agent          claude
✓ gh             gh version 2.79.0
✓ remote         origin → git@github.com:you/project.git
✓ target         main (auto-detected)

all checks passed — run: flawless
```

You need `git` and an agent CLI (`claude` or `codex`) on your `PATH`.
`gh` is optional — without it the PR and CI steps are skipped, everything
else still works.

## 3. Run the gate

There is no `init`, no hook, no extra remote. On any feature branch with
committed work:

```sh
flawless
```

flawless creates a disposable worktree from your branch, then runs the
pipeline: **sync → review → test → lint → push → pr**.

- Test and lint commands are auto-detected (Makefile targets, `go.mod`,
  `package.json`, `Cargo.toml`, `pyproject.toml`) — or set them explicitly
  in [`.flawless.yaml`](/flawless/guides/configuration/).
- When the review finds something blocking, the run pauses and asks you:
  `[f]ix [a]ccept [s]kip step [q]uit`.
- When all gates pass, the branch is pushed and a PR is opened. If the
  pipeline added fix commits, your local branch is fast-forwarded to match
  whenever that is safe.

## 4. Useful variations

```sh
flawless --intent "add rate limiting to the search API"   # guides the review, titles the PR
flawless --yes                # non-interactive: fix what's safe, fail on the rest
flawless --detach             # run in the background…
flawless logs -f              # …and follow it
flawless --skip review,lint   # skip steps this once
flawless status               # what happened in the last run
```

## 5. Optional: write a config

```sh
flawless init
```

This writes a fully commented `.flawless.yaml` where every value is the
default — the file changes nothing until you uncomment something. Most
repos only ever set two lines:

```yaml
commands:
  test: make test
  lint: make lint
```

That's the whole tour. For what each step does, read
[Pipeline](/flawless/concepts/pipeline/).
