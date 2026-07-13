// Package gitx wraps the git CLI. Flawless shells out to the user's git
// rather than embedding a git implementation: it is smaller, always agrees
// with the user's config/credentials, and keeps this package trivial.
package gitx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Git runs git with args in dir and returns trimmed stdout.
func Git(dir string, args ...string) (string, error) {
	return gitEnv(dir, nil, args...)
}

func gitEnv(dir string, extraEnv []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// LC_ALL=C forces untranslated git output: callers match on message
	// text (lease refusals, rebase conflicts), which a localized git
	// (LANG=it_IT, …) would otherwise translate out from under them.
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	cmd.Env = append(cmd.Env, extraEnv...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = strings.TrimSpace(out.String())
		}
		return strings.TrimSpace(out.String()), fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(out.String()), nil
}

// RepoRoot returns the top-level directory of the repo containing dir.
func RepoRoot(dir string) (string, error) {
	return Git(dir, "rev-parse", "--show-toplevel")
}

// CurrentBranch returns the checked-out branch, or an error when detached.
func CurrentBranch(dir string) (string, error) {
	out, err := Git(dir, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("not on a branch (detached HEAD?)")
	}
	return out, nil
}

// HeadSHA returns the full SHA of HEAD.
func HeadSHA(dir string) (string, error) {
	return Git(dir, "rev-parse", "HEAD")
}

// IsDirty reports whether the working tree has uncommitted changes
// (staged, unstaged or untracked).
func IsDirty(dir string) bool {
	out, err := Git(dir, "status", "--porcelain")
	return err == nil && out != ""
}

// HasRemote reports whether the named remote exists.
func HasRemote(dir, remote string) bool {
	_, err := Git(dir, "remote", "get-url", remote)
	return err == nil
}

// RemoteURL returns the URL of the named remote.
func RemoteURL(dir, remote string) (string, error) {
	return Git(dir, "remote", "get-url", remote)
}

// DefaultBranch returns the remote's default branch (e.g. "main").
func DefaultBranch(dir, remote string) (string, error) {
	if out, err := Git(dir, "symbolic-ref", "--short", "refs/remotes/"+remote+"/HEAD"); err == nil {
		return strings.TrimPrefix(out, remote+"/"), nil
	}
	// The remote HEAD ref may not exist locally yet; ask the remote.
	if out, err := Git(dir, "ls-remote", "--symref", remote, "HEAD"); err == nil {
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, "ref:") {
				fields := strings.Fields(line) // ref: refs/heads/main HEAD
				if len(fields) >= 2 {
					return strings.TrimPrefix(fields[1], "refs/heads/"), nil
				}
			}
		}
	}
	for _, cand := range []string{"main", "master"} {
		if _, err := Git(dir, "rev-parse", "--verify", remote+"/"+cand); err == nil {
			return cand, nil
		}
	}
	return "", fmt.Errorf("cannot determine default branch of %s", remote)
}

// AddWorktree creates a detached worktree at path checked out to ref.
func AddWorktree(repoDir, path, ref string) error {
	_, err := Git(repoDir, "worktree", "add", "--detach", path, ref)
	return err
}

// RemoveWorktree removes the worktree at path, discarding its state.
func RemoveWorktree(repoDir, path string) error {
	_, err := Git(repoDir, "worktree", "remove", "--force", path)
	return err
}

// Fetch updates remote-tracking refs.
func Fetch(dir, remote string) error {
	_, err := Git(dir, "fetch", "--quiet", remote)
	return err
}

// Rebase rebases the worktree's HEAD onto upstream. On conflict the rebase
// is aborted and conflicted=true is returned with the error.
func Rebase(dir, upstream string) (conflicted bool, err error) {
	_, err = Git(dir, "rebase", upstream)
	if err == nil {
		return false, nil
	}
	if strings.Contains(err.Error(), "CONFLICT") || strings.Contains(err.Error(), "could not apply") {
		_, _ = Git(dir, "rebase", "--abort")
		return true, err
	}
	_, _ = Git(dir, "rebase", "--abort")
	return false, err
}

// CommitAll stages everything and commits with msg. Returns false when
// there was nothing to commit.
func CommitAll(dir, msg string) (bool, error) {
	if _, err := Git(dir, "add", "-A"); err != nil {
		return false, err
	}
	if out, err := Git(dir, "status", "--porcelain"); err != nil || out == "" {
		return false, err
	}
	_, err := Git(dir, "commit", "-m", msg)
	return err == nil, err
}

// Push pushes sha to remote branch, requiring the remote ref to still be
// exactly at expected ("" = the branch must not exist yet). This is
// --force-with-lease with an explicit expectation rather than the
// remote-tracking ref, so commits someone else pushed at any point the
// run did not incorporate can never be overwritten — not even ones a
// later fetch already made visible locally.
//
// FLAWLESS_INTERNAL=1 lets the push pass a `flawless guard` pre-push hook.
func Push(dir, remote, branch, sha, expected string) error {
	_, err := gitEnv(dir, []string{"FLAWLESS_INTERNAL=1"},
		"push", remote, sha+":refs/heads/"+branch,
		"--force-with-lease=refs/heads/"+branch+":"+expected)
	return err
}

// Diff returns the diff of range base...HEAD, excluding ignored prefixes.
func Diff(dir, base string, ignore []string) (string, error) {
	args := []string{"diff", base + "...HEAD"}
	if len(ignore) > 0 {
		args = append(args, "--")
		args = append(args, ".")
		for _, p := range ignore {
			args = append(args, ":(exclude)"+p)
		}
	}
	return Git(dir, args...)
}

// Subjects returns the commit subjects of base..HEAD, newest first.
func Subjects(dir, base string) ([]string, error) {
	out, err := Git(dir, "log", "--format=%s", base+"..HEAD")
	if err != nil || out == "" {
		return nil, err
	}
	return strings.Split(out, "\n"), nil
}

// IsAncestor reports whether a is an ancestor of b.
func IsAncestor(dir, a, b string) bool {
	_, err := Git(dir, "merge-base", "--is-ancestor", a, b)
	return err == nil
}

// FastForward moves the currently checked-out branch in dir forward to sha
// using merge --ff-only. The caller must ensure the tree is clean.
func FastForward(dir, sha string) error {
	_, err := Git(dir, "merge", "--ff-only", sha)
	return err
}
