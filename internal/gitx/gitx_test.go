package gitx

import (
	"os"
	"path/filepath"
	"testing"
)

// newRepo creates a repo with one commit on main and a bare "origin"
// remote it has pushed to.
func newRepo(t *testing.T) (work, bare string) {
	t.Helper()
	work, bare = t.TempDir(), t.TempDir()
	mustGit := func(dir string, args ...string) string {
		t.Helper()
		out, err := Git(dir, args...)
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
		return out
	}
	mustGit(bare, "init", "--bare", "--initial-branch=main")
	mustGit(work, "init", "--initial-branch=main")
	mustGit(work, "config", "user.email", "t@t")
	mustGit(work, "config", "user.name", "t")
	os.WriteFile(filepath.Join(work, "a.txt"), []byte("one\n"), 0o644)
	mustGit(work, "add", ".")
	mustGit(work, "commit", "-m", "initial")
	mustGit(work, "remote", "add", "origin", bare)
	mustGit(work, "push", "origin", "main")
	return work, bare
}

func TestBasics(t *testing.T) {
	work, _ := newRepo(t)
	if b, err := CurrentBranch(work); err != nil || b != "main" {
		t.Fatalf("branch=%q err=%v", b, err)
	}
	if IsDirty(work) {
		t.Error("fresh repo should be clean")
	}
	os.WriteFile(filepath.Join(work, "b.txt"), []byte("x"), 0o644)
	if !IsDirty(work) {
		t.Error("untracked file should count as dirty")
	}
	if !HasRemote(work, "origin") || HasRemote(work, "nope") {
		t.Error("HasRemote wrong")
	}
}

func TestDefaultBranch(t *testing.T) {
	work, _ := newRepo(t)
	if _, err := Git(work, "fetch", "origin"); err != nil {
		t.Fatal(err)
	}
	b, err := DefaultBranch(work, "origin")
	if err != nil || b != "main" {
		t.Fatalf("default branch = %q, err=%v", b, err)
	}
}

func TestWorktreeAndCommit(t *testing.T) {
	work, _ := newRepo(t)
	wt := filepath.Join(t.TempDir(), "wt")
	sha, _ := HeadSHA(work)
	if err := AddWorktree(work, wt, sha); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(wt, "c.txt"), []byte("fix\n"), 0o644)
	committed, err := CommitAll(wt, "auto-fix")
	if err != nil || !committed {
		t.Fatalf("committed=%v err=%v", committed, err)
	}
	committed, err = CommitAll(wt, "empty")
	if err != nil || committed {
		t.Fatalf("second commit should be a no-op, committed=%v err=%v", committed, err)
	}
	if err := RemoveWorktree(work, wt); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Error("worktree dir should be gone")
	}
}

func TestPushAndAncestry(t *testing.T) {
	work, bare := newRepo(t)
	if _, err := Git(work, "checkout", "-b", "feature"); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(work, "f.txt"), []byte("feat\n"), 0o644)
	if _, err := CommitAll(work, "feature work"); err != nil {
		t.Fatal(err)
	}
	sha, _ := HeadSHA(work)
	// New branch: expected "" means the ref must not exist yet.
	if err := Push(work, "origin", "feature", sha, ""); err != nil {
		t.Fatal(err)
	}
	got, err := Git(bare, "rev-parse", "feature")
	if err != nil || got != sha {
		t.Fatalf("bare feature = %q, want %q (err=%v)", got, sha, err)
	}
	if !IsAncestor(work, "main", "feature") {
		t.Error("main should be ancestor of feature")
	}
	if IsAncestor(work, "feature", "main") {
		t.Error("feature must not be ancestor of main")
	}
}

func TestRebaseConflict(t *testing.T) {
	work, _ := newRepo(t)
	if _, err := Git(work, "checkout", "-b", "feature"); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(work, "a.txt"), []byte("feature version\n"), 0o644)
	if _, err := CommitAll(work, "feature edit"); err != nil {
		t.Fatal(err)
	}
	if _, err := Git(work, "checkout", "main"); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(work, "a.txt"), []byte("main version\n"), 0o644)
	if _, err := CommitAll(work, "main edit"); err != nil {
		t.Fatal(err)
	}
	if _, err := Git(work, "push", "origin", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := Git(work, "checkout", "feature"); err != nil {
		t.Fatal(err)
	}
	if err := Fetch(work, "origin"); err != nil {
		t.Fatal(err)
	}
	conflicted, err := Rebase(work, "origin/main")
	if !conflicted || err == nil {
		t.Fatalf("expected conflict, got conflicted=%v err=%v", conflicted, err)
	}
	// The abort must leave the tree usable.
	if out, err := Git(work, "status", "--porcelain"); err != nil || out != "" {
		t.Errorf("tree not clean after aborted rebase: %q %v", out, err)
	}
}

func TestPushLeaseRefusesUnseenRemoteCommits(t *testing.T) {
	work, bare := newRepo(t)
	if _, err := Git(work, "checkout", "-b", "feature"); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(work, "f.txt"), []byte("v1\n"), 0o644)
	if _, err := CommitAll(work, "v1"); err != nil {
		t.Fatal(err)
	}
	sha1, _ := HeadSHA(work)
	if err := Push(work, "origin", "feature", sha1, ""); err != nil {
		t.Fatal(err)
	}

	// A teammate pushes on top of feature from their own clone.
	mate := t.TempDir()
	if _, err := Git(mate, "clone", "--quiet", bare, "."); err != nil {
		t.Fatal(err)
	}
	Git(mate, "config", "user.email", "m@m")
	Git(mate, "config", "user.name", "m")
	if _, err := Git(mate, "checkout", "feature"); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(mate, "mate.txt"), []byte("their work\n"), 0o644)
	if _, err := CommitAll(mate, "teammate work"); err != nil {
		t.Fatal(err)
	}
	mateSHA, _ := HeadSHA(mate)
	if _, err := Git(mate, "push", "origin", "feature"); err != nil {
		t.Fatal(err)
	}

	// Our push still expects sha1 — even after fetching (which makes the
	// teammate's commit visible locally), the lease must refuse.
	if err := Fetch(work, "origin"); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(work, "f.txt"), []byte("v2\n"), 0o644)
	if _, err := CommitAll(work, "v2"); err != nil {
		t.Fatal(err)
	}
	sha2, _ := HeadSHA(work)
	if err := Push(work, "origin", "feature", sha2, sha1); err == nil {
		t.Fatal("push must be refused: the remote moved past the expected SHA")
	}
	got, _ := Git(bare, "rev-parse", "feature")
	if got != mateSHA {
		t.Fatalf("teammate's commit was lost: bare at %s, want %s", got, mateSHA)
	}
}
