package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config is the effective flawless configuration after merging defaults,
// the global file (~/.config/flawless/config.yaml) and the repo file
// (.flawless.yaml). Everything is optional: with no config files at all,
// flawless auto-detects the agent, the base branch and the test/lint
// commands.
type Config struct {
	// Agent selects the AI agent CLI: "auto" (default), "claude", "codex",
	// "none", or a custom command template containing {prompt_file}.
	Agent string
	// Target is the base branch to rebase onto and open the PR against.
	// Empty means auto-detect the default branch of the remote.
	Target string
	// Remote is the git remote to push to. Default "origin".
	Remote string

	Commands Commands
	Steps    Steps
	AutoFix  AutoFix
	PR       PR

	// Ignore lists path prefixes excluded from the review diff.
	Ignore []string
	// DocsInstructions is extra guidance passed to the agent in the docs step.
	DocsInstructions string
}

// Commands holds the repo's check commands. Empty values are auto-detected
// from the repo layout (go.mod, package.json, ...) or, failing that, chosen
// by the agent.
type Commands struct {
	Test string
	Lint string
}

// Steps toggles pipeline steps. Sync and push always run.
type Steps struct {
	Review bool
	Test   bool
	Lint   bool
	Docs   bool // opt-in
	PR     bool
	CI     bool // opt-in
}

// AutoFix caps how many automatic agent fix attempts each step may make
// before pausing for a human (or failing under --yes).
type AutoFix struct {
	Review int
	Test   int
	Lint   int
}

// PR controls pull request creation.
type PR struct {
	Draft bool
	Base  string // default: Target
}

// Default returns the built-in configuration.
func Default() Config {
	return Config{
		Agent:  "auto",
		Remote: "origin",
		Steps:  Steps{Review: true, Test: true, Lint: true, PR: true},
		AutoFix: AutoFix{
			Review: 1,
			Test:   3,
			Lint:   3,
		},
	}
}

// Load builds the effective config for a repository rooted at repoDir.
func Load(repoDir string) (Config, error) {
	cfg := Default()
	if home, err := os.UserHomeDir(); err == nil {
		globalPath := filepath.Join(home, ".config", "flawless", "config.yaml")
		if err := applyFile(&cfg, globalPath); err != nil {
			return cfg, fmt.Errorf("global config: %w", err)
		}
	}
	if err := applyFile(&cfg, filepath.Join(repoDir, ".flawless.yaml")); err != nil {
		return cfg, fmt.Errorf(".flawless.yaml: %w", err)
	}
	return cfg, nil
}

func applyFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	m, err := parseYAML(string(data))
	if err != nil {
		return err
	}
	return apply(cfg, m)
}

// apply overlays parsed YAML values onto cfg. Unknown keys are an error so
// typos surface immediately instead of being silently ignored.
func apply(cfg *Config, m map[string]any) error {
	for k, v := range m {
		switch k {
		case "agent":
			cfg.Agent = str(v)
		case "target":
			cfg.Target = str(v)
		case "remote":
			cfg.Remote = str(v)
		case "ignore":
			cfg.Ignore = strList(v)
		case "docs_instructions":
			cfg.DocsInstructions = str(v)
		case "commands":
			sub, err := asMap(k, v)
			if err != nil {
				return err
			}
			for sk, sv := range sub {
				switch sk {
				case "test":
					cfg.Commands.Test = str(sv)
				case "lint":
					cfg.Commands.Lint = str(sv)
				default:
					return fmt.Errorf("unknown key commands.%s", sk)
				}
			}
		case "steps":
			sub, err := asMap(k, v)
			if err != nil {
				return err
			}
			for sk, sv := range sub {
				b := boolean(sv)
				switch sk {
				case "review":
					cfg.Steps.Review = b
				case "test":
					cfg.Steps.Test = b
				case "lint":
					cfg.Steps.Lint = b
				case "docs":
					cfg.Steps.Docs = b
				case "pr":
					cfg.Steps.PR = b
				case "ci":
					cfg.Steps.CI = b
				default:
					return fmt.Errorf("unknown key steps.%s", sk)
				}
			}
		case "auto_fix":
			sub, err := asMap(k, v)
			if err != nil {
				return err
			}
			for sk, sv := range sub {
				n := integer(sv)
				switch sk {
				case "review":
					cfg.AutoFix.Review = n
				case "test":
					cfg.AutoFix.Test = n
				case "lint":
					cfg.AutoFix.Lint = n
				default:
					return fmt.Errorf("unknown key auto_fix.%s", sk)
				}
			}
		case "pr":
			sub, err := asMap(k, v)
			if err != nil {
				return err
			}
			for sk, sv := range sub {
				switch sk {
				case "draft":
					cfg.PR.Draft = boolean(sv)
				case "base":
					cfg.PR.Base = str(sv)
				default:
					return fmt.Errorf("unknown key pr.%s", sk)
				}
			}
		default:
			return fmt.Errorf("unknown key %q", k)
		}
	}
	return nil
}

func asMap(key string, v any) (map[string]any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: expected a nested block", key)
	}
	return m, nil
}

func str(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int64:
		return fmt.Sprintf("%d", t)
	}
	return ""
}

func boolean(v any) bool {
	b, _ := v.(bool)
	return b
}

func integer(v any) int {
	if i, ok := v.(int64); ok {
		return int(i)
	}
	return 0
}

func strList(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, str(it))
	}
	return out
}
