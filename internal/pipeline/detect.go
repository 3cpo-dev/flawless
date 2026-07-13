package pipeline

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// detectCommand guesses the repo's test or lint command from its layout.
// kind is "test" or "lint". Returns "" when nothing sensible is found —
// the step is then skipped rather than inventing a wrong command.
func detectCommand(dir, kind string) string {
	if cmd := makefileTarget(dir, kind); cmd != "" {
		return cmd
	}
	switch {
	case exists(dir, "go.mod"):
		if kind == "test" {
			return "go test ./..."
		}
		return "go vet ./..."
	case exists(dir, "package.json"):
		return packageJSONScript(dir, kind)
	case exists(dir, "Cargo.toml"):
		if kind == "test" {
			return "cargo test"
		}
		if _, err := exec.LookPath("cargo-clippy"); err == nil {
			return "cargo clippy --quiet -- -D warnings"
		}
		return "cargo check --quiet"
	case exists(dir, "pyproject.toml") || exists(dir, "setup.py"):
		if kind == "test" {
			if _, err := exec.LookPath("pytest"); err == nil {
				return "pytest -q"
			}
			return ""
		}
		if _, err := exec.LookPath("ruff"); err == nil {
			return "ruff check ."
		}
		return ""
	}
	return ""
}

func exists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

var makeTargetRe = regexp.MustCompile(`(?m)^([A-Za-z0-9_-]+):`)

// makefileTarget returns "make <kind>" when the Makefile defines that target.
func makefileTarget(dir, kind string) string {
	b, err := os.ReadFile(filepath.Join(dir, "Makefile"))
	if err != nil {
		return ""
	}
	for _, m := range makeTargetRe.FindAllStringSubmatch(string(b), -1) {
		if m[1] == kind {
			return "make " + kind
		}
	}
	return ""
}

// packageJSONScript returns "npm run <script>" for a matching script.
// The default npm "test" placeholder (exit 1) is ignored.
func packageJSONScript(dir, kind string) string {
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return ""
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(b, &pkg) != nil {
		return ""
	}
	script, ok := pkg.Scripts[kind]
	if !ok || strings.Contains(script, "no test specified") {
		return ""
	}
	return "npm run " + kind + " --silent"
}
