// Package run persists pipeline run state as plain JSON files under
// .git/flawless/runs/. That's the whole database: no daemon, no SQLite,
// no sockets. `flawless status` and `flawless logs` just read files.
package run

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Statuses a run can be in.
const (
	StatusRunning = "running"
	StatusPassed  = "passed"
	StatusFailed  = "failed"
	StatusAborted = "aborted"
)

// StepRecord captures one executed step.
type StepRecord struct {
	Name    string  `json:"name"`
	Status  string  `json:"status"` // ok | skipped | failed
	Detail  string  `json:"detail,omitempty"`
	Seconds float64 `json:"seconds"`
}

// Run is the persisted state of one pipeline run.
type Run struct {
	ID         string       `json:"id"`
	Branch     string       `json:"branch"`
	Target     string       `json:"target"`
	Remote     string       `json:"remote"`
	Intent     string       `json:"intent,omitempty"`
	StartSHA   string       `json:"start_sha"`
	FinalSHA   string       `json:"final_sha,omitempty"`
	Status     string       `json:"status"`
	Error      string       `json:"error,omitempty"`
	Steps      []StepRecord `json:"steps"`
	PID        int          `json:"pid"`
	PRURL      string       `json:"pr_url,omitempty"`
	StartedAt  time.Time    `json:"started_at"`
	FinishedAt time.Time    `json:"finished_at,omitempty"`

	dir string
}

// Dir returns the runs directory for a repo, creating it if needed.
func Dir(repoDir string) (string, error) {
	d := filepath.Join(repoDir, ".git", "flawless", "runs")
	return d, os.MkdirAll(d, 0o755)
}

// New creates and persists a fresh run record.
func New(repoDir, branch, target, remote, intent, startSHA string) (*Run, error) {
	d, err := Dir(repoDir)
	if err != nil {
		return nil, err
	}
	id := fmt.Sprintf("%s-%s", time.Now().Format("20060102-150405"), sanitize(branch))
	r := &Run{
		ID:        id,
		Branch:    branch,
		Target:    target,
		Remote:    remote,
		Intent:    intent,
		StartSHA:  startSHA,
		Status:    StatusRunning,
		PID:       os.Getpid(),
		StartedAt: time.Now(),
		dir:       d,
	}
	return r, r.Save()
}

// Save writes the run record atomically.
func (r *Run) Save() error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.Path() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, r.Path())
}

// Path returns the JSON file path for this run.
func (r *Run) Path() string { return filepath.Join(r.dir, r.ID+".json") }

// LogPath returns the log file path for this run.
func (r *Run) LogPath() string { return filepath.Join(r.dir, r.ID+".log") }

// Finish marks the run done and persists it.
func (r *Run) Finish(status, errMsg string) {
	r.Status = status
	r.Error = errMsg
	r.FinishedAt = time.Now()
	_ = r.Save()
}

// RecordStep appends a step outcome and persists.
func (r *Run) RecordStep(name, status, detail string, d time.Duration) {
	r.Steps = append(r.Steps, StepRecord{Name: name, Status: status, Detail: detail, Seconds: d.Seconds()})
	_ = r.Save()
}

// List returns runs for the repo, newest first, up to limit (0 = all).
func List(repoDir string, limit int) ([]*Run, error) {
	d, err := Dir(repoDir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(d)
	if err != nil {
		return nil, err
	}
	var runs []*Run
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(d, e.Name()))
		if err != nil {
			continue
		}
		var r Run
		if json.Unmarshal(b, &r) != nil {
			continue
		}
		r.dir = d
		runs = append(runs, &r)
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].StartedAt.After(runs[j].StartedAt) })
	if limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}

// Latest returns the most recent run, or nil when none exist.
func Latest(repoDir string) (*Run, error) {
	runs, err := List(repoDir, 1)
	if err != nil || len(runs) == 0 {
		return nil, err
	}
	return runs[0], nil
}

// Alive reports whether the run's recorded process still exists. A run
// left in "running" whose process is gone simply crashed — no recovery
// daemon needed, the next status call reports it honestly.
func (r *Run) Alive() bool {
	if r.Status != StatusRunning || r.PID == 0 {
		return false
	}
	p, err := os.FindProcess(r.PID)
	if err != nil {
		return false
	}
	return p.Signal(nil) == nil
}

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			return r
		}
		return '-'
	}, s)
}
