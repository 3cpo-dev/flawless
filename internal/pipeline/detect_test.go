package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectCommandGo(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module x\n")
	if got := detectCommand(dir, "test"); got != "go test ./..." {
		t.Errorf("test = %q", got)
	}
	if got := detectCommand(dir, "lint"); got != "go vet ./..." {
		t.Errorf("lint = %q", got)
	}
}

func TestDetectCommandMakefileWins(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module x\n")
	write(t, dir, "Makefile", "build:\n\tgo build\n\ntest:\n\tgo test ./...\n")
	if got := detectCommand(dir, "test"); got != "make test" {
		t.Errorf("Makefile target should win, got %q", got)
	}
	if got := detectCommand(dir, "lint"); got != "go vet ./..." {
		t.Errorf("missing Makefile target should fall through, got %q", got)
	}
}

func TestDetectCommandPackageJSON(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{"scripts": {"test": "vitest run"}}`)
	if got := detectCommand(dir, "test"); got != "npm run test --silent" {
		t.Errorf("test = %q", got)
	}
	if got := detectCommand(dir, "lint"); got != "" {
		t.Errorf("absent script must yield empty, got %q", got)
	}
	// npm's default placeholder script must not count as a test command.
	write(t, dir, "package.json", `{"scripts": {"test": "echo \"Error: no test specified\" && exit 1"}}`)
	if got := detectCommand(dir, "test"); got != "" {
		t.Errorf("placeholder script must be ignored, got %q", got)
	}
}

func TestDetectCommandNothing(t *testing.T) {
	if got := detectCommand(t.TempDir(), "test"); got != "" {
		t.Errorf("empty dir should detect nothing, got %q", got)
	}
}

func TestTailAndFirstLine(t *testing.T) {
	if got := tail("a\nb\nc\nd\n", 2); got != "c\nd" {
		t.Errorf("tail = %q", got)
	}
	if got := firstLine("hello\nworld"); got != "hello" {
		t.Errorf("firstLine = %q", got)
	}
}
