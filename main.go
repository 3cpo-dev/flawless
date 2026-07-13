// Command flawless is a pre-push quality gate: it validates your branch in
// a disposable worktree (agent review, tests, lint, optional docs), then
// pushes and opens the PR — one binary, no daemon, no setup.
package main

import (
	"fmt"
	"os"
)

// version is stamped by -ldflags at release time.
var version = "dev"

const usage = `flawless — ship branches through a quality gate, without the ceremony

Usage:
  flawless [flags]          validate the current branch, push it and open a PR
  flawless init             write a commented .flawless.yaml (optional; flawless
                            works with zero config)
  flawless guard on|off     make the gate mandatory: a pre-push hook refuses
                            direct 'git push' to the gated remote
  flawless status           show the latest run (or --all for history)
  flawless logs [-f]        show the latest run's log
  flawless doctor           check git, agent, gh and repo prerequisites
  flawless version          print the version

Run flags:
  --intent <text>   what this branch is meant to do (guides review; used as PR title)
  --yes             non-interactive: auto-fix what is safe, fail on the rest
  --skip <steps>    comma-separated steps to skip (sync,review,test,lint,docs,push,pr,ci)
  --detach          run in the background; follow with 'flawless logs -f'
  --json            emit machine-readable JSON events (for coding agents)
  --target <branch> base branch (default: the remote's default branch)
  --remote <name>   git remote to push to (default: origin)
  --agent <spec>    agent override: auto | claude | codex | none | custom command

Configuration (all optional): .flawless.yaml in the repo,
~/.config/flawless/config.yaml globally. See 'flawless init'.
`

func main() {
	args := os.Args[1:]
	var err error
	cmd := ""
	if len(args) > 0 && args[0][0] != '-' {
		cmd = args[0]
		args = args[1:]
	}
	switch cmd {
	case "":
		err = cmdRun(args)
	case "init":
		err = cmdInit()
	case "guard":
		err = cmdGuard(args)
	case "status", "runs":
		err = cmdStatus(cmd == "runs", args)
	case "logs":
		err = cmdLogs(args)
	case "doctor":
		err = cmdDoctor()
	case "version":
		fmt.Println("flawless", version)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "flawless: unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "flawless:", err)
		os.Exit(1)
	}
}
