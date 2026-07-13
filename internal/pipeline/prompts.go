package pipeline

import (
	"fmt"
	"strings"

	"github.com/3cpo-dev/flawless/internal/agent"
)

const findingsSchema = `Respond with ONLY a JSON object, no prose, in this exact shape:
{"findings": [{"severity": "blocker|warning|info", "file": "path", "line": 0, "issue": "what is wrong", "fix": "how to fix it", "auto_fixable": true}]}
Severity guide:
- "blocker": bugs, broken behavior, security problems, changes that contradict the stated intent.
- "warning": real but non-blocking issues (unclear naming, missing edge-case handling worth a look).
- "info": observations. Use sparingly.
Set "auto_fixable" true only for mechanical, low-risk fixes. An empty findings list means the diff is clean.`

func reviewPrompt(intent, diff string) string {
	var b strings.Builder
	b.WriteString("You are the code-review gate of the flawless pipeline. Review the following diff before it is pushed.\n")
	if intent != "" {
		b.WriteString("\nThe author's stated intent:\n" + intent + "\n")
	}
	b.WriteString(`
Judge only the diff: correctness, security, data loss, silent behavior changes, and mismatch with the intent. Do not nitpick style that a linter would catch. Do not modify any files.

`)
	b.WriteString(findingsSchema)
	b.WriteString("\n\n--- DIFF ---\n")
	b.WriteString(diff)
	return b.String()
}

func fixPrompt(intent string, blockers []agent.Finding) string {
	var b strings.Builder
	b.WriteString("You are the auto-fix pass of the flawless pipeline, working in a disposable git worktree. Fix the blocking review findings below with minimal, targeted edits. Do not refactor beyond the fix. Do not run git commands; flawless commits for you.\n")
	if intent != "" {
		b.WriteString("\nThe author's intent (do not change it):\n" + intent + "\n")
	}
	b.WriteString("\nFindings to fix:\n")
	for i, f := range blockers {
		loc := f.File
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		fmt.Fprintf(&b, "%d. [%s] %s — %s", i+1, loc, f.Issue, f.Fix)
		b.WriteString("\n")
	}
	return b.String()
}

func checkFixPrompt(step, cmdline, output string) string {
	return fmt.Sprintf(`You are the auto-fix pass of the flawless pipeline, working in a disposable git worktree. The %s command failed.

Command: %s

Output (tail):
%s

Fix the underlying problem with minimal edits. Never delete or weaken a test to make it pass unless the test itself is provably wrong — say so in a code comment if you do. Do not run git commands; flawless commits for you. You may re-run the command above to verify.`, step, cmdline, output)
}

func docsPrompt(intent, diff, instructions string) string {
	var b strings.Builder
	b.WriteString("You are the documentation pass of the flawless pipeline, working in a disposable git worktree. Update any documentation (README, docs/, code comments describing public behavior) that this diff makes stale. Only touch documentation; never change code. If nothing is stale, change nothing.\n")
	if intent != "" {
		b.WriteString("\nThe author's intent:\n" + intent + "\n")
	}
	if instructions != "" {
		b.WriteString("\nRepo documentation instructions:\n" + instructions + "\n")
	}
	b.WriteString("\n--- DIFF ---\n")
	b.WriteString(diff)
	return b.String()
}
