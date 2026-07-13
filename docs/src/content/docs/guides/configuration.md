---
title: Configuration
description: The two optional config files, what belongs in each, and the recommended minimal setup.
---

Configuration is optional. With no files at all, flawless auto-detects
the agent, the target branch and the test/lint commands. Configure only
what detection gets wrong.

## Files

| File | Scope | Typical content |
| --- | --- | --- |
| `.flawless.yaml` (repo root) | travels with the codebase | commands, ignore patterns, docs step |
| `~/.config/flawless/config.yaml` | this machine | preferred agent |

The repo file overrides the global file, field by field. Unknown keys
are **errors**, not warnings — a typo like `agnet:` fails immediately
instead of being silently ignored.

Generate a fully commented template (every value shown is the default):

```sh
flawless init
```

## The recommended minimal setup

Ninety percent of repos need exactly this:

```yaml
# .flawless.yaml
commands:
  test: make test
  lint: make lint
```

Explicit commands make the gate deterministic. Leave them out and
flawless detects them from `Makefile` targets, `go.mod`,
`package.json`, `Cargo.toml` or `pyproject.toml` — detection is decent,
but explicit is better.

## Everything else

```yaml
agent: auto            # auto | claude | codex | none | custom cmd with {prompt_file}
target: main           # base branch; default: remote's default branch
remote: origin

steps:
  review: true
  test: true
  lint: true
  docs: false          # opt-in: agent refreshes docs the diff made stale
  pr: true
  ci: false            # opt-in: watch CI checks after the PR

auto_fix:              # per-step agent fix budgets (0 disables)
  review: 1
  test: 3
  lint: 3

pr:
  draft: false
  base: ""             # default: target

ignore:                # path prefixes excluded from the review diff
  - vendor/
  - dist/

docs_instructions: |
  Docs live in docs/. Keep the CLI reference in sync with --help output.
```

Full field-by-field detail: [Configuration reference](/flawless/reference/configuration/).

## YAML subset

flawless parses its config with a built-in reader (this is how the
binary stays dependency-free). Supported: `key: value`, two-level
nesting with 2-space indents, quoted strings, `true/false`, integers,
lists (block `- item` and flow `[a, b]`), and `#` comments. Not
supported: anchors, multi-line scalars, tabs. `flawless doctor` tells
you immediately if the file doesn't parse.
