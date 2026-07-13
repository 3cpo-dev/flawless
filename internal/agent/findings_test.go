package agent

import "testing"

func TestParseFindingsCleanJSON(t *testing.T) {
	out := `{"findings": [{"severity": "blocker", "file": "a.go", "line": 3, "issue": "nil deref", "auto_fixable": true}]}`
	fs, ok := ParseFindings(out)
	if !ok || len(fs) != 1 {
		t.Fatalf("ok=%v findings=%v", ok, fs)
	}
	f := fs[0]
	if !f.Blocking() || f.File != "a.go" || f.Line != 3 || !f.AutoFixable {
		t.Errorf("finding mismatch: %+v", f)
	}
}

func TestParseFindingsFencedAndProse(t *testing.T) {
	out := "Here is my review:\n```json\n{\"findings\": [{\"severity\": \"warning\", \"issue\": \"unclear name\"}]}\n```\nHope that helps!"
	fs, ok := ParseFindings(out)
	if !ok || len(fs) != 1 || fs[0].Severity != SeverityWarning {
		t.Fatalf("ok=%v findings=%v", ok, fs)
	}
}

func TestParseFindingsBareArray(t *testing.T) {
	fs, ok := ParseFindings(`[{"severity": "info", "issue": "note"}]`)
	if !ok || len(fs) != 1 || fs[0].Severity != SeverityInfo {
		t.Fatalf("ok=%v findings=%v", ok, fs)
	}
}

func TestParseFindingsEmptyList(t *testing.T) {
	fs, ok := ParseFindings(`{"findings": []}`)
	if !ok || len(fs) != 0 {
		t.Fatalf("clean diff should parse as ok with zero findings, got ok=%v %v", ok, fs)
	}
}

func TestParseFindingsGarbage(t *testing.T) {
	if _, ok := ParseFindings("The diff looks fine to me!"); ok {
		t.Fatal("prose without JSON must not parse")
	}
}

func TestParseFindingsNormalizes(t *testing.T) {
	out := `{"findings": [{"severity": "critical", "issue": "boom"}, {"severity": "info", "issue": "  "}]}`
	fs, ok := ParseFindings(out)
	if !ok || len(fs) != 1 {
		t.Fatalf("want 1 finding after dropping empty issue, got %v", fs)
	}
	if fs[0].Severity != SeverityWarning {
		t.Errorf("unknown severity should normalize to warning, got %q", fs[0].Severity)
	}
}

func TestDetect(t *testing.T) {
	if _, err := Detect("custom command without placeholder"); err == nil {
		t.Error("custom command without {prompt_file} must be rejected")
	}
	a, err := Detect("cat {prompt_file}")
	if err != nil || a.Name != "custom" {
		t.Errorf("custom detect failed: %v %v", a, err)
	}
	a, err = Detect("none")
	if err != nil || a != nil {
		t.Errorf("none should return nil agent, got %v %v", a, err)
	}
}
