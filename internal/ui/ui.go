// Package ui renders pipeline progress. There is deliberately no TUI:
// output is plain lines that read well in a terminal, in CI logs, and to
// coding agents. With JSON mode enabled every event is additionally a
// machine-readable JSON line, which is the whole "agent API".
package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ANSI codes, disabled when not a TTY or NO_COLOR is set.
const (
	cReset = "\x1b[0m"
	cDim   = "\x1b[2m"
	cBold  = "\x1b[1m"
	cRed   = "\x1b[31m"
	cGreen = "\x1b[32m"
	cYell  = "\x1b[33m"
	cCyan  = "\x1b[36m"
)

// UI writes human output to Out and, when JSON is true, JSON event lines
// to Out as well (one object per line). Log, when set, receives an
// uncolored copy of everything for the run log file.
type UI struct {
	Out   io.Writer
	In    io.Reader
	Log   io.Writer
	JSON  bool
	Color bool
	// Yes makes Ask return its non-interactive default instead of prompting.
	Yes bool
}

// New builds a UI writing to stdout/stdin with color auto-detection.
func New() *UI {
	color := os.Getenv("NO_COLOR") == "" && isTTY(os.Stdout)
	return &UI{Out: os.Stdout, In: os.Stdin, Color: color}
}

func isTTY(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func (u *UI) paint(code, s string) string {
	if !u.Color {
		return s
	}
	return code + s + cReset
}

func (u *UI) emit(human string, event map[string]any) {
	if u.JSON {
		event["ts"] = time.Now().UTC().Format(time.RFC3339)
		b, _ := json.Marshal(event)
		fmt.Fprintln(u.Out, string(b))
	} else if human != "" {
		fmt.Fprintln(u.Out, human)
	}
	if u.Log != nil && human != "" {
		fmt.Fprintln(u.Log, human)
	}
}

// StepStart announces a pipeline step.
func (u *UI) StepStart(name, detail string) {
	line := fmt.Sprintf("%s %s", u.paint(cCyan, "▸ "+name), u.paint(cDim, detail))
	u.emit(line, map[string]any{"event": "step_start", "step": name, "detail": detail})
}

// StepEnd reports a step outcome: "ok", "skipped" or "failed".
func (u *UI) StepEnd(name, status, detail string, d time.Duration) {
	mark, col := "✓", cGreen
	switch status {
	case "skipped":
		mark, col = "○", cDim
	case "failed":
		mark, col = "✗", cRed
	}
	dur := ""
	if d > 0 {
		dur = u.paint(cDim, fmt.Sprintf(" (%s)", d.Round(100*time.Millisecond)))
	}
	line := fmt.Sprintf("%s %s  %s%s", u.paint(col, mark), name, detail, dur)
	u.emit(line, map[string]any{"event": "step_end", "step": name, "status": status, "detail": detail, "seconds": d.Seconds()})
}

// Info prints a neutral message.
func (u *UI) Info(msg string) {
	u.emit("  "+msg, map[string]any{"event": "info", "message": msg})
}

// Warn prints a warning.
func (u *UI) Warn(msg string) {
	u.emit("  "+u.paint(cYell, "! ")+msg, map[string]any{"event": "warning", "message": msg})
}

// Error prints an error message.
func (u *UI) Error(msg string) {
	u.emit(u.paint(cRed, "error: ")+msg, map[string]any{"event": "error", "message": msg})
}

// Header prints the run banner.
func (u *UI) Header(msg string) {
	u.emit(u.paint(cBold, msg), map[string]any{"event": "run_start", "message": msg})
}

// Finding prints one agent finding.
func (u *UI) Finding(severity, location, issue string) {
	col := cDim
	switch severity {
	case "blocker":
		col = cRed
	case "warning":
		col = cYell
	}
	loc := ""
	if location != "" {
		loc = u.paint(cDim, location+" ")
	}
	line := fmt.Sprintf("    %s %s%s", u.paint(col, "["+severity+"]"), loc, issue)
	u.emit(line, map[string]any{"event": "finding", "severity": severity, "location": location, "issue": issue})
}

// Ask prompts the user to choose one of options (single letters, e.g.
// "f" fix, "a" approve, "s" skip, "q" quit). Under --yes or JSON mode it
// returns def without prompting.
func (u *UI) Ask(question string, options []string, labels map[string]string, def string) string {
	if u.Yes || u.JSON || !isInteractive(u.In) {
		u.Info(fmt.Sprintf("%s → auto: %s", question, labels[def]))
		return def
	}
	var parts []string
	for _, o := range options {
		parts = append(parts, fmt.Sprintf("[%s]%s", o, strings.TrimPrefix(labels[o], o)))
	}
	fmt.Fprintf(u.Out, "  %s %s ", u.paint(cBold, question), u.paint(cDim, strings.Join(parts, " ")))
	sc := bufio.NewScanner(u.In)
	for sc.Scan() {
		ans := strings.ToLower(strings.TrimSpace(sc.Text()))
		if ans == "" {
			return def
		}
		for _, o := range options {
			if ans == o || ans == labels[o] {
				return o
			}
		}
		fmt.Fprintf(u.Out, "  please answer one of %s: ", strings.Join(options, "/"))
	}
	return def
}

func isInteractive(r io.Reader) bool {
	f, ok := r.(*os.File)
	return ok && isTTY(f)
}
