package main

import (
	"fmt"

	"github.com/3cpo-dev/flawless/internal/config"
	"github.com/3cpo-dev/flawless/internal/gitx"
	"github.com/3cpo-dev/flawless/internal/guard"
)

func cmdGuard(args []string) error {
	sub := "status"
	if len(args) > 0 {
		sub = args[0]
	}
	repoDir, err := gitx.RepoRoot(".")
	if err != nil {
		return fmt.Errorf("not inside a git repository")
	}
	cfg, err := config.Load(repoDir)
	if err != nil {
		return err
	}
	switch sub {
	case "on":
		if err := guard.Install(repoDir, cfg.Remote); err != nil {
			return err
		}
		fmt.Printf("guard on: direct 'git push %s …' is now refused in this repo\n", cfg.Remote)
		fmt.Println("ship through the gate with: flawless   (bypass once: FLAWLESS_BYPASS=1 git push …)")
		return nil
	case "off":
		if err := guard.Remove(repoDir); err != nil {
			return err
		}
		fmt.Println("guard off: direct pushes are allowed again")
		return nil
	case "status":
		on, err := guard.Installed(repoDir)
		if err != nil {
			return err
		}
		if on {
			fmt.Printf("guard is ON — direct pushes to %q are refused (flawless guard off to disable)\n", cfg.Remote)
		} else {
			fmt.Println("guard is off — the gate is voluntary (flawless guard on to enforce it)")
		}
		return nil
	default:
		return fmt.Errorf("unknown guard subcommand %q (use: on, off, status)", sub)
	}
}
