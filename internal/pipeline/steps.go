package pipeline

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/3cpo-dev/flawless/internal/agent"
	"github.com/3cpo-dev/flawless/internal/gitx"
)

// --- sync -----------------------------------------------------------------

func stepSync(c *Context) (string, string, error) {
	c.UI.StepStart("sync", "fetching "+c.Remote+" and rebasing onto "+c.Upstream())
	if err := gitx.Fetch(c.Worktree, c.Remote); err != nil {
		return "", "", err
	}
	if _, err := gitx.Git(c.Worktree, "rev-parse", "--verify", c.Upstream()); err != nil {
		return "skipped", c.Upstream() + " does not exist yet", nil
	}
	before, _ := gitx.HeadSHA(c.Worktree)
	conflicted, err := gitx.Rebase(c.Worktree, c.Upstream())
	if conflicted {
		return "", "", fmt.Errorf("rebase onto %s hit conflicts; rebase manually and re-run flawless", c.Upstream())
	}
	if err != nil {
		return "", "", err
	}
	after, _ := gitx.HeadSHA(c.Worktree)
	if before == after {
		return "ok", "already up to date with " + c.Upstream(), nil
	}
	return "ok", "rebased onto " + c.Upstream(), nil
}

// --- review ---------------------------------------------------------------

func stepReview(c *Context) (string, string, error) {
	c.UI.StepStart("review", "agent code review of the diff")
	diff, err := gitx.Diff(c.Worktree, c.Upstream(), c.Cfg.Ignore)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(diff) == "" {
		return "skipped", "no changes against " + c.Upstream(), nil
	}
	attempts := 0
	for {
		out, err := c.Agent.Run(c.Ctx, c.Worktree, reviewPrompt(c.Intent, diff), false)
		if err != nil {
			return "", "", err
		}
		findings, ok := agent.ParseFindings(out)
		if !ok {
			c.UI.Warn("agent review returned no parsable findings; treating as clean")
			return "ok", "no findings (unparsed output)", nil
		}
		blockers := show(c, findings)
		if len(blockers) == 0 {
			return "ok", fmt.Sprintf("%d findings, none blocking", len(findings)), nil
		}
		outcome, err := resolveGate(c, "review", blockers, attempts < c.Cfg.AutoFix.Review)
		if err != nil {
			return "", "", err
		}
		switch outcome {
		case gateProceed:
			return "ok", fmt.Sprintf("proceeding with %d blocker(s) accepted", len(blockers)), nil
		case gateFix:
			attempts++
			if err := applyAgentFix(c, "review", fixPrompt(c.Intent, blockers), blockers); err != nil {
				return "", "", err
			}
			diff, err = gitx.Diff(c.Worktree, c.Upstream(), c.Cfg.Ignore)
			if err != nil {
				return "", "", err
			}
			continue // re-review the updated diff
		case gateFail:
			return "", "", fmt.Errorf("%d blocking finding(s) unresolved", len(blockers))
		}
	}
}

// show prints findings and returns the blocking ones.
func show(c *Context, findings []agent.Finding) []agent.Finding {
	var blockers []agent.Finding
	for _, f := range findings {
		loc := f.File
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		c.UI.Finding(f.Severity, loc, f.Issue)
		if f.Blocking() {
			blockers = append(blockers, f)
		}
	}
	return blockers
}

// --- test / lint ----------------------------------------------------------

func stepTest(c *Context) (string, string, error) {
	cmd := c.Cfg.Commands.Test
	if cmd == "" {
		cmd = detectCommand(c.Worktree, "test")
	}
	return checkStep(c, "test", cmd, c.Cfg.AutoFix.Test)
}

func stepLint(c *Context) (string, string, error) {
	cmd := c.Cfg.Commands.Lint
	if cmd == "" {
		cmd = detectCommand(c.Worktree, "lint")
	}
	return checkStep(c, "lint", cmd, c.Cfg.AutoFix.Lint)
}

// checkStep runs a shell check command with an agent auto-fix loop.
func checkStep(c *Context, name, cmd string, fixBudget int) (string, string, error) {
	if cmd == "" {
		return "skipped", "no " + name + " command configured or detected", nil
	}
	c.UI.StepStart(name, cmd)
	attempts := 0
	for {
		out, err := shell(c, cmd)
		if err == nil {
			if attempts > 0 {
				return "ok", fmt.Sprintf("passing after %d auto-fix(es)", attempts), nil
			}
			return "ok", "passed", nil
		}
		c.UI.Warn(fmt.Sprintf("%s failed: %s", name, firstLine(tail(out, 1))))
		canFix := c.Agent != nil && attempts < fixBudget
		if canFix {
			attempts++
			c.UI.Info(fmt.Sprintf("auto-fix attempt %d/%d", attempts, fixBudget))
			if err := applyAgentFix(c, name, checkFixPrompt(name, cmd, tail(out, 120)), nil); err != nil {
				return "", "", err
			}
			continue
		}
		outcome, gerr := resolveGate(c, name, []agent.Finding{{
			Severity: agent.SeverityBlocker,
			Issue:    fmt.Sprintf("%s command failed: %s", name, cmd),
		}}, false)
		if gerr != nil {
			return "", "", gerr
		}
		switch outcome {
		case gateProceed:
			return "ok", name + " failure accepted by user", nil
		case gateFix:
			attempts = 0 // user granted another fix round
			fixBudget = 1
			continue
		default:
			return "", "", fmt.Errorf("%s failed (%s); last output:\n%s", name, cmd, tail(out, 25))
		}
	}
}

// --- docs -----------------------------------------------------------------

func stepDocs(c *Context) (string, string, error) {
	c.UI.StepStart("docs", "agent documentation pass")
	diff, err := gitx.Diff(c.Worktree, c.Upstream(), c.Cfg.Ignore)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(diff) == "" {
		return "skipped", "no changes to document", nil
	}
	if _, err := c.Agent.Run(c.Ctx, c.Worktree, docsPrompt(c.Intent, diff, c.Cfg.DocsInstructions), true); err != nil {
		return "", "", err
	}
	committed, err := gitx.CommitAll(c.Worktree, "docs: update for "+c.Branch+" (flawless)")
	if err != nil {
		return "", "", err
	}
	if !committed {
		return "ok", "documentation already adequate", nil
	}
	return "ok", "documentation updated", nil
}

// --- push -----------------------------------------------------------------

func stepPush(c *Context) (string, string, error) {
	c.UI.StepStart("push", "pushing validated branch to "+c.Remote)
	sha, err := gitx.HeadSHA(c.Worktree)
	if err != nil {
		return "", "", err
	}
	c.Run.FinalSHA = sha
	remoteRef := c.Remote + "/" + c.Branch
	if cur, err := gitx.Git(c.Worktree, "rev-parse", "--verify", remoteRef); err == nil && cur == sha {
		return "skipped", "remote already at " + sha[:10], nil
	}
	// force-with-lease is only needed when the rebase rewrote history.
	lease := false
	if _, err := gitx.Git(c.Worktree, "rev-parse", "--verify", remoteRef); err == nil {
		lease = !gitx.IsAncestor(c.Worktree, remoteRef, sha)
	}
	if err := gitx.Push(c.Worktree, c.Remote, c.Branch, sha, lease); err != nil {
		return "", "", err
	}
	syncLocalBranch(c, sha)
	return "ok", fmt.Sprintf("pushed %s to %s/%s", sha[:10], c.Remote, c.Branch), nil
}

// syncLocalBranch fast-forwards the developer's checked-out branch to the
// validated SHA when that is safe, so their local view matches the remote.
func syncLocalBranch(c *Context, sha string) {
	branch, err := gitx.CurrentBranch(c.RepoDir)
	if err != nil || branch != c.Branch {
		return
	}
	if gitx.IsDirty(c.RepoDir) || !gitx.IsAncestor(c.RepoDir, "HEAD", sha) {
		c.UI.Info(fmt.Sprintf("local branch not updated (pipeline rewrote history or tree is dirty); run: git pull --rebase %s %s", c.Remote, c.Branch))
		return
	}
	if err := gitx.FastForward(c.RepoDir, sha); err == nil {
		c.UI.Info("local branch fast-forwarded to validated commit")
	}
}

// --- pr -------------------------------------------------------------------

func stepPR(c *Context) (string, string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "skipped", "gh CLI not installed", nil
	}
	c.UI.StepStart("pr", "creating or updating the pull request")
	base := c.Cfg.PR.Base
	if base == "" {
		base = c.Target
	}
	if url, err := ghOut(c, "pr", "view", c.Branch, "--json", "url", "--jq", ".url"); err == nil && url != "" {
		c.Run.PRURL = url
		return "ok", "PR already open, updated by push: " + url, nil
	}
	title := c.Intent
	if title == "" {
		if subj, _ := gitx.Subjects(c.Worktree, c.Upstream()); len(subj) > 0 {
			title = subj[len(subj)-1] // oldest commit subject
		} else {
			title = c.Branch
		}
	}
	args := []string{"pr", "create", "--head", c.Branch, "--base", base, "--title", title, "--body", prBody(c)}
	if c.Cfg.PR.Draft {
		args = append(args, "--draft")
	}
	url, err := ghOut(c, args...)
	if err != nil {
		return "", "", fmt.Errorf("gh pr create: %v", err)
	}
	c.Run.PRURL = strings.TrimSpace(url)
	return "ok", c.Run.PRURL, nil
}

func prBody(c *Context) string {
	var b strings.Builder
	if c.Intent != "" {
		b.WriteString(c.Intent + "\n\n")
	}
	b.WriteString("Validated by [flawless](https://github.com/3cpo-dev/flawless): ")
	var passed []string
	for _, s := range c.Run.Steps {
		if s.Status == "ok" {
			passed = append(passed, s.Name)
		}
	}
	b.WriteString(strings.Join(passed, ", "))
	b.WriteString(".\n")
	return b.String()
}

// --- ci -------------------------------------------------------------------

func stepCI(c *Context) (string, string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "skipped", "gh CLI not installed", nil
	}
	c.UI.StepStart("ci", "waiting for CI checks on the pushed branch")
	out, err := ghOut(c, "pr", "checks", c.Branch, "--watch", "--interval", "15")
	if err != nil {
		return "", "", fmt.Errorf("CI checks failed:\n%s", tail(out, 20))
	}
	return "ok", "all checks green", nil
}

// --- shared helpers ---------------------------------------------------------

// applyAgentFix asks the agent to fix issues in the worktree and commits
// whatever it changed.
func applyAgentFix(c *Context, step, prompt string, _ []agent.Finding) error {
	if c.Agent == nil {
		return fmt.Errorf("auto-fix requested but agent is 'none'")
	}
	if _, err := c.Agent.Run(c.Ctx, c.Worktree, prompt, true); err != nil {
		return err
	}
	committed, err := gitx.CommitAll(c.Worktree, fmt.Sprintf("fix: %s findings (flawless auto-fix)", step))
	if err != nil {
		return err
	}
	if !committed {
		c.UI.Warn("agent made no changes")
	}
	return nil
}

type gateOutcome int

const (
	gateProceed gateOutcome = iota // accept and continue
	gateFix                        // attempt an agent fix
	gateFail                       // stop the run
)

// resolveGate decides what to do about blocking findings. Interactive runs
// ask the user; --yes/JSON runs auto-fix when allowed and fixable,
// otherwise fail — flawless never silently ships a blocker.
func resolveGate(c *Context, step string, blockers []agent.Finding, fixAllowed bool) (gateOutcome, error) {
	if c.Yes || c.UI.JSON {
		if fixAllowed && c.Agent != nil && allFixable(blockers) {
			return gateFix, nil
		}
		return gateFail, nil
	}
	opts := []string{"f", "a", "s", "q"}
	labels := map[string]string{"f": "fix", "a": "accept", "s": "skip step", "q": "quit"}
	if !fixAllowed || c.Agent == nil {
		opts = []string{"a", "s", "q"}
	}
	q := fmt.Sprintf("%d blocking finding(s) in %s — what now?", len(blockers), step)
	switch c.UI.Ask(q, opts, labels, opts[0]) {
	case "f":
		return gateFix, nil
	case "a", "s":
		return gateProceed, nil
	default:
		return gateFail, ErrAborted
	}
}

func allFixable(fs []agent.Finding) bool {
	for _, f := range fs {
		if !f.AutoFixable {
			return false
		}
	}
	return len(fs) > 0
}

// shell runs a command line in the worktree via sh -c.
func shell(c *Context, cmdline string) (string, error) {
	cmd := exec.CommandContext(c.Ctx, "sh", "-c", cmdline)
	cmd.Dir = c.Worktree
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

func ghOut(c *Context, args ...string) (string, error) {
	cmd := exec.CommandContext(c.Ctx, "gh", args...)
	cmd.Dir = c.Worktree
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("%v: %s", err, strings.TrimSpace(errb.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// tail returns the last n lines of s.
func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
