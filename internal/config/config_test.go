package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseYAMLSubset(t *testing.T) {
	src := `
# top comment
agent: claude
target: "main"   # trailing comment
remote: 'origin'
commands:
  test: go test ./...
  lint: golangci-lint run
steps:
  docs: true
  ci: false
auto_fix:
  test: 5
ignore:
  - vendor/
  - dist/
pr:
  draft: yes
`
	m, err := parseYAML(src)
	if err != nil {
		t.Fatal(err)
	}
	if m["agent"] != "claude" || m["target"] != "main" || m["remote"] != "origin" {
		t.Errorf("scalars wrong: %v", m)
	}
	cmds := m["commands"].(map[string]any)
	if cmds["test"] != "go test ./..." || cmds["lint"] != "golangci-lint run" {
		t.Errorf("commands wrong: %v", cmds)
	}
	steps := m["steps"].(map[string]any)
	if steps["docs"] != true || steps["ci"] != false {
		t.Errorf("steps wrong: %v", steps)
	}
	if m["auto_fix"].(map[string]any)["test"] != int64(5) {
		t.Errorf("auto_fix wrong: %v", m["auto_fix"])
	}
	want := []any{"vendor/", "dist/"}
	if !reflect.DeepEqual(m["ignore"], want) {
		t.Errorf("ignore = %v, want %v", m["ignore"], want)
	}
	if m["pr"].(map[string]any)["draft"] != true {
		t.Errorf("pr wrong: %v", m["pr"])
	}
}

func TestParseYAMLFlowList(t *testing.T) {
	m, err := parseYAML("ignore: [vendor/, \"a b\", 3]")
	if err != nil {
		t.Fatal(err)
	}
	want := []any{"vendor/", "a b", int64(3)}
	if !reflect.DeepEqual(m["ignore"], want) {
		t.Errorf("got %v, want %v", m["ignore"], want)
	}
}

func TestParseYAMLErrors(t *testing.T) {
	for _, src := range []string{
		"\tagent: claude", // tab indentation
		"- orphan item",   // list without key
		"just some words", // no colon
	} {
		if _, err := parseYAML(src); err == nil {
			t.Errorf("expected error for %q", src)
		}
	}
}

func TestLoadMergesRepoOverGlobal(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	globalDir := filepath.Join(home, ".config", "flawless")
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(globalDir, "config.yaml"), []byte("agent: codex\nsteps:\n  docs: true\n"), 0o644)
	os.WriteFile(filepath.Join(repo, ".flawless.yaml"), []byte("agent: claude\ncommands:\n  test: make check\n"), 0o644)

	cfg, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Agent != "claude" {
		t.Errorf("repo should override global agent, got %q", cfg.Agent)
	}
	if !cfg.Steps.Docs {
		t.Error("global steps.docs=true should survive")
	}
	if cfg.Commands.Test != "make check" {
		t.Errorf("commands.test = %q", cfg.Commands.Test)
	}
	if cfg.AutoFix.Test != 3 {
		t.Errorf("default auto_fix.test = %d, want 3", cfg.AutoFix.Test)
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	os.WriteFile(filepath.Join(repo, ".flawless.yaml"), []byte("agnet: claude\n"), 0o644)
	if _, err := Load(repo); err == nil {
		t.Fatal("expected error for unknown key 'agnet'")
	}
}

func TestDefaultsWithNoFiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Agent != "auto" || cfg.Remote != "origin" || !cfg.Steps.Review || cfg.Steps.Docs {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}
