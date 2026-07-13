package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/3cpo-dev/flawless/internal/gitx"
	"github.com/3cpo-dev/flawless/internal/run"
)

func cmdStatus(all bool, args []string) error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	allFlag := fs.Bool("all", false, "list all runs")
	limit := fs.Int("limit", 20, "max runs to list with --all")
	if err := fs.Parse(args); err != nil {
		return err
	}
	all = all || *allFlag

	repoDir, err := gitx.RepoRoot(".")
	if err != nil {
		return fmt.Errorf("not inside a git repository")
	}

	if all {
		runs, err := run.List(repoDir, *limit)
		if err != nil {
			return err
		}
		if len(runs) == 0 {
			fmt.Println("no runs yet — start one with: flawless")
			return nil
		}
		for _, r := range runs {
			fmt.Printf("%-32s %-20s %-8s %s\n", r.ID, r.Branch, effectiveStatus(r), stepsSummary(r))
		}
		return nil
	}

	r, err := run.Latest(repoDir)
	if err != nil {
		return err
	}
	if r == nil {
		fmt.Println("no runs yet — start one with: flawless")
		return nil
	}
	fmt.Printf("run:      %s\n", r.ID)
	fmt.Printf("branch:   %s → %s/%s\n", r.Branch, r.Remote, r.Target)
	fmt.Printf("status:   %s\n", effectiveStatus(r))
	if r.Error != "" {
		fmt.Printf("error:    %s\n", r.Error)
	}
	if r.PRURL != "" {
		fmt.Printf("pr:       %s\n", r.PRURL)
	}
	fmt.Printf("started:  %s\n", r.StartedAt.Format(time.RFC1123))
	if !r.FinishedAt.IsZero() {
		fmt.Printf("duration: %s\n", r.FinishedAt.Sub(r.StartedAt).Round(time.Second))
	}
	for _, s := range r.Steps {
		fmt.Printf("  %-8s %-8s %s\n", s.Name, s.Status, s.Detail)
	}
	return nil
}

// effectiveStatus reports "crashed" for runs recorded as running whose
// process no longer exists.
func effectiveStatus(r *run.Run) string {
	if r.Status == run.StatusRunning && !r.Alive() {
		return "crashed"
	}
	return r.Status
}

func stepsSummary(r *run.Run) string {
	ok := 0
	for _, s := range r.Steps {
		if s.Status == "ok" {
			ok++
		}
	}
	return fmt.Sprintf("%d/%d steps ok", ok, len(r.Steps))
}
