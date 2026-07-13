// Package agent invokes AI coding agent CLIs (Claude Code, Codex, or any
// custom command) and parses their structured output. Flawless has no
// bundled model and no API keys: it drives whatever agent the developer
// already uses, inside the isolated worktree.
package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CallTimeout bounds a single agent invocation.
const CallTimeout = 15 * time.Minute

// Agent is a runnable AI agent CLI.
type Agent struct {
	// Name is "claude", "codex" or "custom".
	Name string
	// custom holds the user-supplied command template when Name=="custom".
	custom string
}

// Detect resolves the configured agent to a runnable one.
// spec is "auto", "claude", "codex", "none", or a custom command template
// containing "{prompt_file}".
func Detect(spec string) (*Agent, error) {
	spec = strings.TrimSpace(spec)
	switch spec {
	case "", "auto":
		for _, name := range []string{"claude", "codex"} {
			if _, err := exec.LookPath(name); err == nil {
				return &Agent{Name: name}, nil
			}
		}
		return nil, fmt.Errorf("no agent CLI found (looked for: claude, codex); set 'agent' in .flawless.yaml or install one")
	case "none":
		return nil, nil
	case "claude", "codex":
		if _, err := exec.LookPath(spec); err != nil {
			return nil, fmt.Errorf("agent %q is configured but not on PATH", spec)
		}
		return &Agent{Name: spec}, nil
	default:
		if !strings.Contains(spec, "{prompt_file}") {
			return nil, fmt.Errorf("custom agent command must contain {prompt_file}: %q", spec)
		}
		return &Agent{Name: "custom", custom: spec}, nil
	}
}

// Run executes the agent in dir with prompt. When write is true the agent
// is allowed to edit files in dir (it is always a disposable worktree).
// Returns the agent's final text output.
func (a *Agent) Run(ctx context.Context, dir, prompt string, write bool) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, CallTimeout)
	defer cancel()

	promptFile := filepath.Join(dir, ".flawless-prompt.md")
	if err := os.WriteFile(promptFile, []byte(prompt), 0o600); err != nil {
		return "", err
	}
	defer os.Remove(promptFile)

	var cmd *exec.Cmd
	switch a.Name {
	case "claude":
		args := []string{"-p", prompt, "--output-format", "text"}
		if write {
			args = append(args, "--permission-mode", "acceptEdits")
		}
		cmd = exec.CommandContext(ctx, "claude", args...)
	case "codex":
		args := []string{"exec"}
		if write {
			args = append(args, "--sandbox", "workspace-write")
		}
		args = append(args, prompt)
		cmd = exec.CommandContext(ctx, "codex", args...)
	case "custom":
		line := strings.ReplaceAll(a.custom, "{prompt_file}", promptFile)
		cmd = exec.CommandContext(ctx, "sh", "-c", line)
	default:
		return "", fmt.Errorf("unknown agent %q", a.Name)
	}
	cmd.Dir = dir
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("agent %s timed out after %s", a.Name, CallTimeout)
		}
		msg := strings.TrimSpace(errb.String())
		if len(msg) > 400 {
			msg = msg[:400] + "…"
		}
		return out.String(), fmt.Errorf("agent %s failed: %v: %s", a.Name, err, msg)
	}
	return out.String(), nil
}
