---
title: Installation
description: Install flawless on macOS, Linux or Windows — installer script, go install, or from source.
---

flawless is a single static binary. Installing it means putting that one
file on your `PATH`; there is no daemon to register, no service files, no
post-install steps.

## macOS / Linux (installer script)

```sh
curl -fsSL https://raw.githubusercontent.com/3cpo-dev/flawless/main/install.sh | sh
```

The script downloads the latest release for your OS/architecture into
`~/.local/bin/flawless` (or `/usr/local/bin` when `~/.local/bin` is not
on your `PATH` and you run it with sudo). It does nothing else — you can
read it in one screen.

## Windows

Download `flawless_windows_amd64.exe` from the
[latest release](https://github.com/3cpo-dev/flawless/releases/latest),
rename it to `flawless.exe`, and put it somewhere on your `PATH`.

> Note: `--detach` uses Unix process groups and is not supported on
> Windows; run flawless in a second terminal instead.

## go install

If you have Go 1.22+:

```sh
go install github.com/3cpo-dev/flawless@latest
```

Because flawless has zero dependencies and zero telemetry in any build,
this produces exactly the same behavior as a release binary (only the
`flawless version` string differs).

## From source

```sh
git clone https://github.com/3cpo-dev/flawless
cd flawless
make build          # or: go build -o flawless .
./flawless version
```

## Prerequisites

| Tool | Needed for | Required? |
| --- | --- | --- |
| `git` 2.20+ | everything | yes |
| `claude` or `codex` (or any agent CLI) | review, docs, auto-fix | recommended — without one, flawless still runs tests/lint/push |
| `gh` | PR creation, CI watching | optional — those steps skip themselves politely |

Verify with:

```sh
flawless doctor
```

## Updating

Re-run the installer script, or `go install …@latest`. There is no
daemon to restart and no state to migrate — the new binary just replaces
the old one, even mid-workday.

## Uninstalling

```sh
rm "$(command -v flawless)"
```

Per-repo run history lives in `.git/flawless/` and is removed with the
repo (or with `rm -rf .git/flawless`). The optional global config is
`~/.config/flawless/`. That is the complete footprint.
