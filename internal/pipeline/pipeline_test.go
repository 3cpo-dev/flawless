package pipeline

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/3cpo-dev/flawless/internal/agent"
	"github.com/3cpo-dev/flawless/internal/config"
	"github.com/3cpo-dev/flawless/internal/gitx"
	"github.com/3cpo-dev/flawless/internal/run"
	"github.com/3cpo-dev/flawless/internal/ui"
)

// fakeAgent is a shell script standing in for a real agent CLI. It reads
// the prompt file to tell review passes from fix passes:
//   - review: reports a blocker while src.txt contains "BUG", else clean
//   - fix (review findings): rewrites BUG -> FIXED in src.txt
//   - fix (failing check): flips check.sh to exit 0
const fakeAgent = `#!/bin/sh
prompt="$1"
if grep -q "code-review gate" "$prompt"; then
  if grep -q BUG src.txt 2>/dev/null; then
    echo '{"findings": [{"severity": "blocker", "file": "src.txt", "issue": "contains BUG", "fix": "replace BUG with FIXED", "auto_fixable": true}]}'
  else
    echo '{"findings": []}'
  fi
elif grep -q "review findings below" "$prompt"; then
  sed -i.bak 's/BUG/FIXED/' src.txt && rm -f src.txt.bak
elif grep -q "command failed" "$prompt"; then
  printf '#!/bin/sh\nexit 0\n' > check.sh
fi
`

// setup builds: bare origin with main, a work clone on branch "feature"
// with a buggy src.txt and a failing check.sh, and a worktree ready for
// the pipeline.
func setup(t *testing.T) (c *Context, work, bare string, out *bytes.Buffer) {
	t.Helper()
	work, bare = t.TempDir(), t.TempDir()
	g := func(dir string, args ...string) {
		t.Helper()
		if _, err := gitx.Git(dir, args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	g(bare, "init", "--bare", "--initial-branch=main")
	g(work, "init", "--initial-branch=main")
	g(work, "config", "user.email", "t@t")
	g(work, "config", "user.name", "t")
	os.WriteFile(filepath.Join(work, "src.txt"), []byte("hello\n"), 0o644)
	os.WriteFile(filepath.Join(work, "fake-agent.sh"), []byte(fakeAgent), 0o755)
	g(work, "add", ".")
	g(work, "commit", "-m", "initial")
	g(work, "remote", "add", "origin", bare)
	g(work, "push", "origin", "main")

	g(work, "checkout", "-b", "feature")
	os.WriteFile(filepath.Join(work, "src.txt"), []byte("hello BUG\n"), 0o644)
	os.WriteFile(filepath.Join(work, "check.sh"), []byte("#!/bin/sh\ngrep -q FIXED src.txt\n"), 0o755)
	g(work, "add", ".")
	g(work, "commit", "-m", "feature work")
	g(work, "fetch", "origin")

	sha, err := gitx.HeadSHA(work)
	if err != nil {
		t.Fatal(err)
	}
	wt := filepath.Join(t.TempDir(), "wt")
	if err := gitx.AddWorktree(work, wt, sha); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gitx.RemoveWorktree(work, wt) })

	cfg := config.Default()
	cfg.Agent = "sh ./fake-agent.sh {prompt_file}"
	cfg.Commands.Test = "sh ./check.sh"
	cfg.Steps.Lint = false
	cfg.Steps.PR = false

	ag, err := agent.Detect(cfg.Agent)
	if err != nil {
		t.Fatal(err)
	}
	r, err := run.New(work, "feature", "main", "origin", "make src great", sha)
	if err != nil {
		t.Fatal(err)
	}
	out = &bytes.Buffer{}
	u := &ui.UI{Out: out, In: strings.NewReader(""), Yes: true}
	c = &Context{
		Ctx: context.Background(), Cfg: cfg, UI: u, Agent: ag,
		RepoDir: work, Worktree: wt,
		Branch: "feature", Target: "main", Remote: "origin",
		Intent: "make src great", Run: r, Skip: map[string]bool{}, Yes: true,
	}
	return c, work, bare, out
}

func TestExecuteEndToEnd(t *testing.T) {
	c, _, bare, out := setup(t)
	if err := Execute(c); err != nil {
		t.Fatalf("pipeline failed: %v\noutput:\n%s", err, out.String())
	}
	// The review auto-fix must have landed and been pushed.
	pushed, err := gitx.Git(bare, "rev-parse", "feature")
	if err != nil {
		t.Fatalf("feature not pushed to origin: %v", err)
	}
	if pushed != c.Run.FinalSHA {
		t.Errorf("pushed %s, run recorded %s", pushed, c.Run.FinalSHA)
	}
	content, _ := gitx.Git(c.Worktree, "show", "HEAD:src.txt")
	if !strings.Contains(content, "FIXED") || strings.Contains(content, "BUG") {
		t.Errorf("auto-fix not in pushed history: %q", content)
	}
	for _, s := range c.Run.Steps {
		if s.Status == "failed" {
			t.Errorf("step %s failed: %s", s.Name, s.Detail)
		}
	}
}

func TestExecuteFailsOnUnfixableBlocker(t *testing.T) {
	c, _, _, out := setup(t)
	// Remove the fix budget: under --yes an unfixed blocker must fail.
	c.Cfg.AutoFix.Review = 0
	err := Execute(c)
	if err == nil {
		t.Fatalf("expected failure, output:\n%s", out.String())
	}
	if !strings.Contains(err.Error(), "review") {
		t.Errorf("failure should come from review, got: %v", err)
	}
	if c.Run.FinalSHA != "" {
		t.Error("nothing must be pushed when review blocks")
	}
}

func TestExecuteSkipFlag(t *testing.T) {
	c, _, bare, out := setup(t)
	c.Skip["review"] = true
	// Without review's fix, the test step's own fixer flips check.sh.
	if err := Execute(c); err != nil {
		t.Fatalf("pipeline failed: %v\noutput:\n%s", err, out.String())
	}
	if _, err := gitx.Git(bare, "rev-parse", "feature"); err != nil {
		t.Error("feature should still be pushed")
	}
	for _, s := range c.Run.Steps {
		if s.Name == "review" && s.Status != "skipped" {
			t.Errorf("review should be skipped, got %s", s.Status)
		}
	}
}
