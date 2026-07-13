package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/3cpo-dev/flawless/internal/agent"
	"github.com/3cpo-dev/flawless/internal/config"
	"github.com/3cpo-dev/flawless/internal/gitx"
	"github.com/3cpo-dev/flawless/internal/pipeline"
	"github.com/3cpo-dev/flawless/internal/run"
	"github.com/3cpo-dev/flawless/internal/ui"
)

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	intent := fs.String("intent", "", "what this branch is meant to do")
	yes := fs.Bool("yes", false, "non-interactive mode")
	skip := fs.String("skip", "", "comma-separated steps to skip")
	detach := fs.Bool("detach", false, "run in the background")
	jsonOut := fs.Bool("json", false, "emit JSON events")
	target := fs.String("target", "", "base branch")
	remote := fs.String("remote", "", "git remote")
	agentSpec := fs.String("agent", "", "agent override")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoDir, err := gitx.RepoRoot(".")
	if err != nil {
		return fmt.Errorf("not inside a git repository")
	}
	branch, err := gitx.CurrentBranch(repoDir)
	if err != nil {
		return err
	}
	cfg, err := config.Load(repoDir)
	if err != nil {
		return err
	}
	if *remote != "" {
		cfg.Remote = *remote
	}
	if *target != "" {
		cfg.Target = *target
	}
	if *agentSpec != "" {
		cfg.Agent = *agentSpec
	}
	if !gitx.HasRemote(repoDir, cfg.Remote) {
		return fmt.Errorf("remote %q not found; add it or pass --remote", cfg.Remote)
	}
	if cfg.Target == "" {
		cfg.Target, err = gitx.DefaultBranch(repoDir, cfg.Remote)
		if err != nil {
			return fmt.Errorf("%v; set 'target' in .flawless.yaml or pass --target", err)
		}
	}
	if branch == cfg.Target {
		return fmt.Errorf("you are on %q, the target branch itself; create a feature branch first", branch)
	}
	skipSet, err := parseSkip(*skip)
	if err != nil {
		return err
	}

	if *detach {
		return spawnDetached(args)
	}

	u := ui.New()
	u.JSON = *jsonOut
	u.Yes = *yes
	detached := os.Getenv("FLAWLESS_DETACHED") == "1"
	if detached {
		u.Yes = true
		u.Color = false
	}

	ag, err := agent.Detect(cfg.Agent)
	if err != nil {
		return err
	}
	startSHA, err := gitx.HeadSHA(repoDir)
	if err != nil {
		return err
	}
	if gitx.IsDirty(repoDir) {
		u.Warn("working tree has uncommitted changes; they are NOT part of this run (flawless validates committed work only)")
	}
	if *intent == "" {
		*intent = inferIntent(repoDir, cfg.Remote+"/"+cfg.Target)
	}

	r, err := run.New(repoDir, branch, cfg.Target, cfg.Remote, *intent, startSHA)
	if err != nil {
		return err
	}
	logf, err := os.Create(r.LogPath())
	if err != nil {
		return err
	}
	defer logf.Close()
	u.Log = logf
	if detached {
		u.Out = logf
		u.Log = nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	agentName := "none"
	if ag != nil {
		agentName = ag.Name
	}
	u.Header(fmt.Sprintf("flawless %s → %s/%s  (agent: %s, run: %s)", branch, cfg.Remote, cfg.Target, agentName, r.ID))

	wt := filepath.Join(os.TempDir(), "flawless-"+r.ID)
	if err := gitx.AddWorktree(repoDir, wt, startSHA); err != nil {
		r.Finish(run.StatusFailed, err.Error())
		return err
	}
	defer func() {
		if err := gitx.RemoveWorktree(repoDir, wt); err != nil {
			u.Warn("could not remove worktree " + wt + ": " + err.Error())
		}
	}()

	pctx := &pipeline.Context{
		Ctx: ctx, Cfg: cfg, UI: u, Agent: ag,
		RepoDir: repoDir, Worktree: wt,
		Branch: branch, Target: cfg.Target, Remote: cfg.Remote,
		Intent: *intent, Run: r, Skip: skipSet, Yes: u.Yes,
	}
	if err := pipeline.Execute(pctx); err != nil {
		if ctx.Err() != nil || err == pipeline.ErrAborted {
			r.Finish(run.StatusAborted, err.Error())
			u.Error("run aborted")
		} else {
			r.Finish(run.StatusFailed, err.Error())
			u.Error(err.Error())
			u.Info("details: flawless logs")
		}
		return fmt.Errorf("run %s: %s", r.ID, r.Status)
	}
	r.Finish(run.StatusPassed, "")
	if r.PRURL != "" {
		u.Header("✦ flawless: all gates passed — " + r.PRURL)
	} else {
		u.Header("✦ flawless: all gates passed")
	}
	return nil
}

func parseSkip(s string) (map[string]bool, error) {
	set := map[string]bool{}
	if s == "" {
		return set, nil
	}
	valid := map[string]bool{}
	for _, n := range pipeline.Names() {
		valid[n] = true
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if !valid[part] {
			return nil, fmt.Errorf("--skip: unknown step %q (steps: %s)", part, strings.Join(pipeline.Names(), ","))
		}
		set[part] = true
	}
	return set, nil
}

// inferIntent builds a fallback intent from the branch's commit subjects.
func inferIntent(repoDir, upstream string) string {
	subjects, err := gitx.Subjects(repoDir, upstream)
	if err != nil || len(subjects) == 0 {
		return ""
	}
	if len(subjects) > 5 {
		subjects = subjects[:5]
	}
	return "Commits on this branch: " + strings.Join(subjects, "; ")
}

// spawnDetached re-executes flawless without --detach as a detached
// process. Output goes to the run log; no daemon is involved — the run is
// just a normal background process.
func spawnDetached(args []string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	var kept []string
	for _, a := range args {
		if a != "--detach" && a != "-detach" {
			kept = append(kept, a)
		}
	}
	cmd := exec.Command(self, kept...)
	cmd.Env = append(os.Environ(), "FLAWLESS_DETACHED=1")
	cmd.Stdout, cmd.Stderr, cmd.Stdin = nil, nil, nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	fmt.Printf("flawless: run started in the background (pid %d)\n", cmd.Process.Pid)
	fmt.Println("follow it with:  flawless logs -f")
	return cmd.Process.Release()
}
