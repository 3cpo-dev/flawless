package guard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/3cpo-dev/flawless/internal/gitx"
)

func newRepo(t *testing.T) (work, bare string) {
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
	os.WriteFile(filepath.Join(work, "a.txt"), []byte("one\n"), 0o644)
	g(work, "add", ".")
	g(work, "commit", "-m", "initial")
	g(work, "remote", "add", "origin", bare)
	g(work, "push", "origin", "main")
	return work, bare
}

func commit(t *testing.T, dir, name string) {
	t.Helper()
	os.WriteFile(filepath.Join(dir, name), []byte("x\n"), 0o644)
	if _, err := gitx.CommitAll(dir, "add "+name); err != nil {
		t.Fatal(err)
	}
}

func TestGuardBlocksDirectPushOnly(t *testing.T) {
	work, _ := newRepo(t)
	if err := Install(work, "origin"); err != nil {
		t.Fatal(err)
	}
	on, err := Installed(work)
	if err != nil || !on {
		t.Fatalf("Installed = %v, %v", on, err)
	}

	commit(t, work, "b.txt")
	if _, err := gitx.Git(work, "push", "origin", "main"); err == nil {
		t.Fatal("direct push must be refused while guard is on")
	}

	// flawless's own push (FLAWLESS_INTERNAL=1) passes the hook.
	sha, _ := gitx.HeadSHA(work)
	base, _ := gitx.Git(work, "rev-parse", "origin/main")
	if err := gitx.Push(work, "origin", "main", sha, base); err != nil {
		t.Fatalf("internal push should pass the guard: %v", err)
	}

	commit(t, work, "c.txt")
	if err := Remove(work); err != nil {
		t.Fatal(err)
	}
	if _, err := gitx.Git(work, "push", "origin", "main"); err != nil {
		t.Fatalf("direct push should work after guard off: %v", err)
	}
}

func TestGuardRefusesForeignHook(t *testing.T) {
	work, _ := newRepo(t)
	hooks, _ := gitx.Git(work, "rev-parse", "--git-path", "hooks")
	p := filepath.Join(work, hooks, "pre-push")
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755)

	if err := Install(work, "origin"); err == nil {
		t.Fatal("must not overwrite a pre-push hook it does not own")
	}
	if err := Remove(work); err == nil {
		t.Fatal("must not remove a pre-push hook it does not own")
	}
}
