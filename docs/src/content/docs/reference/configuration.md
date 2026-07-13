---
title: Configuration Reference
description: Every configuration field, its type, default and effect.
---

Two files, same format, repo overrides global field-by-field:

- `~/.config/flawless/config.yaml` — machine scope
- `<repo>/.flawless.yaml` — repo scope

Unknown keys are errors. All fields optional.

## Top level

| Field | Type | Default | Effect |
| --- | --- | --- | --- |
| `agent` | string | `auto` | `auto`, `claude`, `codex`, `none`, or a custom command containing `{prompt_file}` ([details](/flawless/guides/agents/)) |
| `target` | string | remote's default branch | base branch for rebase and PR |
| `remote` | string | `origin` | git remote pushed to |
| `ignore` | list | `[]` | path prefixes excluded from the review diff |
| `docs_instructions` | string | `""` | extra guidance appended to the docs-step prompt |

## `commands`

| Field | Type | Default | Effect |
| --- | --- | --- | --- |
| `commands.test` | string | auto-detected | test command, run with `sh -c` in the worktree |
| `commands.lint` | string | auto-detected | lint command |

Auto-detection order: `Makefile` target with the step's name → language
file (`go.mod` → `go test ./...` / `go vet ./...`; `package.json`
script; `Cargo.toml` → `cargo test` / clippy-or-check;
`pyproject.toml`/`setup.py` → pytest / ruff when installed). Nothing
detected → the step is skipped with that reason.

## `steps`

| Field | Default | Notes |
| --- | --- | --- |
| `steps.review` | `true` | needs an agent |
| `steps.test` | `true` | |
| `steps.lint` | `true` | |
| `steps.docs` | `false` | opt-in; needs an agent |
| `steps.pr` | `true` | needs `gh`; skips itself otherwise |
| `steps.ci` | `false` | opt-in; needs `gh` |

`sync` and `push` are not configurable — a gate that neither syncs nor
publishes isn't a gate. Both can still be skipped for a single run with
`--skip`.

## `auto_fix`

| Field | Default | Meaning |
| --- | --- | --- |
| `auto_fix.review` | `1` | fix rounds for blocking review findings |
| `auto_fix.test` | `3` | fix attempts for a failing test command |
| `auto_fix.lint` | `3` | fix attempts for a failing lint command |

`0` disables automation for that step (gates offer accept/skip/quit only).

## `pr`

| Field | Default | Meaning |
| --- | --- | --- |
| `pr.draft` | `false` | open PRs as drafts |
| `pr.base` | `""` (= `target`) | PR base branch when it differs from the rebase target |

## Full example

```yaml
agent: claude
target: main

commands:
  test: make test
  lint: make lint

steps:
  docs: true

auto_fix:
  test: 2

pr:
  draft: true

ignore:
  - vendor/
  - gen/

docs_instructions: |
  User docs live in docs/. Update the CLI reference when flags change.
```
