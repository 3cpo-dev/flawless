package main

import (
	"fmt"
	"os/exec"

	"github.com/3cpo-dev/flawless/internal/agent"
	"github.com/3cpo-dev/flawless/internal/config"
	"github.com/3cpo-dev/flawless/internal/gitx"
)

func cmdDoctor() error {
	failures := 0
	check := func(name string, ok bool, detail string) {
		mark := "✓"
		if !ok {
			mark = "✗"
			failures++
		}
		fmt.Printf("%s %-14s %s\n", mark, name, detail)
	}

	_, gitErr := exec.LookPath("git")
	check("git", gitErr == nil, versionOf("git", "--version"))

	repoDir, repoErr := gitx.RepoRoot(".")
	if repoErr != nil {
		check("repository", false, "not inside a git repository (repo checks skipped)")
		fmt.Println()
	} else {
		check("repository", true, repoDir)
	}

	cfg := config.Default()
	if repoErr == nil {
		var err error
		cfg, err = config.Load(repoDir)
		check("config", err == nil, configDetail(err))
	}

	ag, agErr := agent.Detect(cfg.Agent)
	switch {
	case agErr != nil:
		check("agent", false, agErr.Error())
	case ag == nil:
		check("agent", true, "none (review/docs/auto-fix disabled by config)")
	default:
		check("agent", true, ag.Name)
	}

	_, ghErr := exec.LookPath("gh")
	check("gh", ghErr == nil, ghDetail(ghErr))

	if repoErr == nil {
		if gitx.HasRemote(repoDir, cfg.Remote) {
			url, _ := gitx.RemoteURL(repoDir, cfg.Remote)
			check("remote", true, cfg.Remote+" → "+url)
			target := cfg.Target
			if target == "" {
				var err error
				target, err = gitx.DefaultBranch(repoDir, cfg.Remote)
				check("target", err == nil, targetDetail(target, err))
			} else {
				check("target", true, target+" (configured)")
			}
		} else {
			check("remote", false, "remote "+cfg.Remote+" not found")
		}
	}

	if failures > 0 {
		return fmt.Errorf("%d check(s) failed", failures)
	}
	fmt.Println("\nall checks passed — run: flawless")
	return nil
}

func versionOf(bin string, arg string) string {
	out, err := exec.Command(bin, arg).Output()
	if err != nil {
		return "not found"
	}
	return firstLine(string(out))
}

func firstLine(s string) string {
	for i, r := range s {
		if r == '\n' {
			return s[:i]
		}
	}
	return s
}

func configDetail(err error) string {
	if err != nil {
		return err.Error()
	}
	return "valid"
}

func ghDetail(err error) string {
	if err != nil {
		return "not installed (pr and ci steps will be skipped)"
	}
	return versionOf("gh", "--version")
}

func targetDetail(target string, err error) string {
	if err != nil {
		return err.Error()
	}
	return target + " (auto-detected)"
}
