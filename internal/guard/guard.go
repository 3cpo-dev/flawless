// Package guard installs an optional pre-push hook that makes the gate
// structural: direct `git push` to the gated remote is refused, so the
// only way a branch ships is through flawless (or an explicit bypass).
// This is the honest answer to "a command you must remember to run can
// be forgotten" — opt-in, one file, no daemon.
package guard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/3cpo-dev/flawless/internal/gitx"
)

const marker = "# flawless guard"

func hookScript(remote string) string {
	return fmt.Sprintf(`#!/bin/sh
%s — pushes to '%s' must go through flawless.
# Installed by 'flawless guard on'; remove with 'flawless guard off'.
[ "$FLAWLESS_INTERNAL" = "1" ] && exit 0
[ "$FLAWLESS_BYPASS" = "1" ] && exit 0
[ "$1" = "%s" ] || exit 0
echo "flawless guard: direct pushes to '%s' are gated." >&2
echo "  validate and push:  flawless" >&2
echo "  bypass this once:   FLAWLESS_BYPASS=1 git push $*" >&2
exit 1
`, marker, remote, remote, remote)
}

// path returns the pre-push hook location, honoring core.hooksPath.
func path(repoDir string) (string, error) {
	out, err := gitx.Git(repoDir, "rev-parse", "--git-path", "hooks")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(out) {
		out = filepath.Join(repoDir, out)
	}
	return filepath.Join(out, "pre-push"), nil
}

// Install writes the guard hook. It refuses to overwrite a pre-push hook
// it does not own.
func Install(repoDir, remote string) error {
	p, err := path(repoDir)
	if err != nil {
		return err
	}
	if existing, err := os.ReadFile(p); err == nil && !strings.Contains(string(existing), marker) {
		return fmt.Errorf("a pre-push hook already exists at %s; merge it manually or remove it first", p)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(hookScript(remote)), 0o755)
}

// Remove deletes the guard hook. Removing a hook flawless does not own
// is refused.
func Remove(repoDir string) error {
	p, err := path(repoDir)
	if err != nil {
		return err
	}
	existing, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !strings.Contains(string(existing), marker) {
		return fmt.Errorf("the pre-push hook at %s was not installed by flawless; not touching it", p)
	}
	return os.Remove(p)
}

// Installed reports whether the guard hook is in place.
func Installed(repoDir string) (bool, error) {
	p, err := path(repoDir)
	if err != nil {
		return false, err
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.Contains(string(b), marker), nil
}
