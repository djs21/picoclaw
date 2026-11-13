package vcc

import (
	"strings"
	"testing"
)

func TestCompileFirstCompaction(t *testing.T) {
	input := CompileInput{
		Messages: []map[string]interface{}{
			{"role": "user", "content": "Fix the login bug"},
			{"role": "assistant", "content": "I will fix it now"},
		},
	}
	got := Compile(input)
	if got == "" {
		t.Fatal("Compile returned empty summary")
	}
	if !strings.Contains(got, "[Session Goal]") {
		t.Errorf("Compile missing session goal:\n%s", got)
	}
	if !strings.HasSuffix(strings.TrimSpace(got), RECALL_NOTE) {
		t.Errorf("Compile missing trailing RECALL_NOTE:\n%s", got)
	}
	if strings.Count(got, RECALL_NOTE) != 1 {
		t.Errorf("Compile RECALL_NOTE count = %d, want 1:\n%s", strings.Count(got, RECALL_NOTE), got)
	}
}

func TestCompileMergesPrevious(t *testing.T) {
	input := CompileInput{
		PreviousSummary: "[Session Goal]\n- fix landing page\n\n---\n\n" + RECALL_NOTE,
		Messages: []map[string]interface{}{
			{"role": "user", "content": "Fix the login bug"},
			{"role": "assistant", "content": "I will fix it now"},
		},
	}
	got := Compile(input)
	if got == "" {
		t.Fatal("Compile returned empty summary")
	}
	if strings.Count(got, RECALL_NOTE) != 1 {
		t.Errorf("merged Compile RECALL_NOTE count = %d, want 1:\n%s", strings.Count(got, RECALL_NOTE), got)
	}
	if !strings.Contains(got, "fix landing page") {
		t.Errorf("merged Compile dropped sticky goal:\n%s", got)
	}
	if !strings.Contains(got, "Fix the login bug") {
		t.Errorf("merged Compile missing fresh goal:\n%s", got)
	}
}
