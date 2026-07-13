package agent

import (
	"encoding/json"
	"strings"
)

// Severity levels a finding can carry. Blockers gate the pipeline;
// warnings and infos are reported but never block.
const (
	SeverityBlocker = "blocker"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

// Finding is one issue reported by the agent.
type Finding struct {
	Severity    string `json:"severity"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Issue       string `json:"issue"`
	Fix         string `json:"fix,omitempty"`
	AutoFixable bool   `json:"auto_fixable"`
}

// Blocking reports whether the finding must be resolved before pushing.
func (f Finding) Blocking() bool { return f.Severity == SeverityBlocker }

type findingsDoc struct {
	Findings []Finding `json:"findings"`
}

// ParseFindings extracts findings from agent output. Agents are asked to
// reply with `{"findings": [...]}` but often wrap JSON in prose or code
// fences, so parsing is deliberately lenient: it tries the whole output,
// then fenced ```json blocks, then the outermost {...} or [...] span.
// A nil slice with ok=false means no JSON was found at all.
func ParseFindings(out string) (findings []Finding, ok bool) {
	candidates := []string{strings.TrimSpace(out)}
	if block := fencedBlock(out); block != "" {
		candidates = append(candidates, block)
	}
	if span := outerSpan(out, '{', '}'); span != "" {
		candidates = append(candidates, span)
	}
	if span := outerSpan(out, '[', ']'); span != "" {
		candidates = append(candidates, span)
	}
	for _, c := range candidates {
		var doc findingsDoc
		if err := json.Unmarshal([]byte(c), &doc); err == nil && doc.Findings != nil {
			return normalize(doc.Findings), true
		}
		var list []Finding
		if err := json.Unmarshal([]byte(c), &list); err == nil {
			return normalize(list), true
		}
	}
	return nil, false
}

func normalize(fs []Finding) []Finding {
	out := fs[:0]
	for _, f := range fs {
		if strings.TrimSpace(f.Issue) == "" {
			continue
		}
		switch f.Severity {
		case SeverityBlocker, SeverityWarning, SeverityInfo:
		default:
			f.Severity = SeverityWarning
		}
		out = append(out, f)
	}
	return out
}

func fencedBlock(s string) string {
	for _, fence := range []string{"```json", "```"} {
		i := strings.Index(s, fence)
		if i < 0 {
			continue
		}
		rest := s[i+len(fence):]
		j := strings.Index(rest, "```")
		if j < 0 {
			continue
		}
		return strings.TrimSpace(rest[:j])
	}
	return ""
}

func outerSpan(s string, open, close byte) string {
	i := strings.IndexByte(s, open)
	j := strings.LastIndexByte(s, close)
	if i < 0 || j <= i {
		return ""
	}
	return strings.TrimSpace(s[i : j+1])
}
