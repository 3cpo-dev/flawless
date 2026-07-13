// Package pipeline runs the flawless validation pipeline inside a
// disposable git worktree. Unlike its inspiration, there is no daemon, no
// gate repository and no hook: the pipeline is an ordinary function call
// in an ordinary process, and everything it needs travels in Context.
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/3cpo-dev/flawless/internal/agent"
	"github.com/3cpo-dev/flawless/internal/config"
	"github.com/3cpo-dev/flawless/internal/run"
	"github.com/3cpo-dev/flawless/internal/ui"
)

// ErrAborted is returned when the user quits at a gate.
var ErrAborted = errors.New("run aborted by user")

// Context carries everything a step needs.
type Context struct {
	Ctx      context.Context
	Cfg      config.Config
	UI       *ui.UI
	Agent    *agent.Agent // nil when agent: none
	RepoDir  string       // main repository root
	Worktree string       // disposable worktree the pipeline works in
	Branch   string
	Target   string // base branch (no remote prefix)
	Remote   string
	Intent   string
	Run      *run.Run
	Skip     map[string]bool // step names to skip, from --skip
	Yes      bool            // non-interactive: fix what's fixable, fail otherwise
}

// Upstream returns the remote-tracking ref of the target branch.
func (c *Context) Upstream() string { return c.Remote + "/" + c.Target }

// step is one pipeline stage. Run returns a status ("ok" | "skipped" |
// "failed"), a one-line detail, and an error only for fatal problems.
type step struct {
	name    string
	enabled func(*Context) bool
	run     func(*Context) (status, detail string, err error)
}

func steps() []step {
	return []step{
		{"sync", func(*Context) bool { return true }, stepSync},
		{"review", func(c *Context) bool { return c.Cfg.Steps.Review && c.Agent != nil }, stepReview},
		{"test", func(c *Context) bool { return c.Cfg.Steps.Test }, stepTest},
		{"lint", func(c *Context) bool { return c.Cfg.Steps.Lint }, stepLint},
		{"docs", func(c *Context) bool { return c.Cfg.Steps.Docs && c.Agent != nil }, stepDocs},
		{"push", func(*Context) bool { return true }, stepPush},
		{"pr", func(c *Context) bool { return c.Cfg.Steps.PR }, stepPR},
		{"ci", func(c *Context) bool { return c.Cfg.Steps.CI }, stepCI},
	}
}

// Names lists all step names in order, for --skip validation and docs.
func Names() []string {
	var out []string
	for _, s := range steps() {
		out = append(out, s.name)
	}
	return out
}

// Execute runs the pipeline to completion. The run record is updated as
// steps finish; the caller owns worktree creation and cleanup.
func Execute(c *Context) error {
	for _, s := range steps() {
		if err := c.Ctx.Err(); err != nil {
			return err
		}
		if c.Skip[s.name] {
			c.UI.StepEnd(s.name, "skipped", "skipped by --skip", 0)
			c.Run.RecordStep(s.name, "skipped", "skipped by --skip", 0)
			continue
		}
		if !s.enabled(c) {
			continue // disabled in config: stay quiet, it's not part of this repo's gate
		}
		start := time.Now()
		status, detail, err := s.run(c)
		dur := time.Since(start)
		if err != nil {
			c.UI.StepEnd(s.name, "failed", firstLine(err.Error()), dur)
			c.Run.RecordStep(s.name, "failed", err.Error(), dur)
			return fmt.Errorf("%s: %w", s.name, err)
		}
		c.UI.StepEnd(s.name, status, detail, dur)
		c.Run.RecordStep(s.name, status, detail, dur)
	}
	return nil
}

func firstLine(s string) string {
	for i, r := range s {
		if r == '\n' {
			return s[:i]
		}
	}
	return s
}
