package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/3cpo-dev/flawless/internal/gitx"
)

// configTemplate is written by `flawless init`. Every value shown is the
// default, commented out: an untouched file changes nothing.
const configTemplate = `# .flawless.yaml — flawless configuration (everything here is optional).
# Uncomment only what differs from the defaults shown.

# AI agent used for review, docs and auto-fix.
# auto = first of: claude, codex. Or a custom command containing {prompt_file}.
#agent: auto

# Base branch to rebase onto and open PRs against (default: remote's default branch).
#target: main

# Git remote to push to.
#remote: origin

# Check commands. Empty = auto-detected from the repo (Makefile targets,
# go.mod, package.json, Cargo.toml, pyproject.toml).
#commands:
#  test: go test ./...
#  lint: golangci-lint run

# Pipeline steps. sync and push always run; docs and ci are opt-in.
#steps:
#  review: true
#  test: true
#  lint: true
#  docs: false
#  pr: true
#  ci: false

# Max automatic agent fix attempts per step before pausing (or failing under --yes).
#auto_fix:
#  review: 1
#  test: 3
#  lint: 3

# Pull request options.
#pr:
#  draft: false
#  base: ""            # default: target

# Path prefixes excluded from the review diff.
#ignore:
#  - vendor/

# Extra guidance for the docs step.
#docs_instructions: ""
`

func cmdInit() error {
	repoDir, err := gitx.RepoRoot(".")
	if err != nil {
		return fmt.Errorf("not inside a git repository")
	}
	path := filepath.Join(repoDir, ".flawless.yaml")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf(".flawless.yaml already exists")
	}
	if err := os.WriteFile(path, []byte(configTemplate), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote .flawless.yaml (fully commented — flawless works with zero config)")
	fmt.Println("check your setup with:  flawless doctor")
	fmt.Println("run the gate with:      flawless")
	return nil
}
