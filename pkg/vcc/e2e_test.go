package vcc

import (
	"strings"
	"testing"
)

func TestE2ECompileSmoke(t *testing.T) {
	messages := []map[string]interface{}{
		{"role": "user", "content": "Implement the VCC context manager"},
		{"role": "assistant", "content": "I will start with the types package"},
		{"role": "user", "content": "Also update the configuration documentation"},
		{"role": "assistant", "content": "Done, docs are updated"},
		{"role": "user", "content": "Now run the package tests"},
		{"role": "assistant", "content": "Running tests now"},
	}

	compile := func(prev string) string {
		blocks := filterNoise(normalize(messages))
		data := buildSections(BuildSectionsInput{Blocks: blocks})
		fresh := formatSummary(data)
		if prev == "" {
			if fresh == "" {
				return ""
			}
			return wrapLongLines(fresh, TUI_SAFE_LINE_CHARS) + SECTION_SEPARATOR + RECALL_NOTE
		}
		merged := mergePrevious(stripRecallNote(prev), fresh)
		if merged == "" {
			return ""
		}
		return wrapLongLines(merged, TUI_SAFE_LINE_CHARS) + SECTION_SEPARATOR + RECALL_NOTE
	}

	first := compile("")
	if first == "" {
		t.Fatal("e2e compile returned empty summary")
	}
	if !strings.Contains(first, "[Session Goal]") {
		t.Errorf("e2e compile missing Session Goal:\n%s", first)
	}
	if !strings.Contains(first, "[user]") {
		t.Errorf("e2e compile missing brief transcript user block:\n%s", first)
	}
	if strings.Count(first, RECALL_NOTE) != 1 {
		t.Errorf("e2e compile RECALL_NOTE count = %d, want 1:\n%s", strings.Count(first, RECALL_NOTE), first)
	}

	second := compile("")
	if second != first {
		t.Error("e2e compile is non-deterministic across identical inputs")
	}

	prev := "[Session Goal]\n- build earlier feature\n\n---\n\n" + RECALL_NOTE
	merged := compile(prev)
	if !strings.Contains(merged, "build earlier feature") {
		t.Errorf("e2e merged compile dropped previous goal:\n%s", merged)
	}
	if !strings.Contains(merged, "Implement the VCC context manager") {
		t.Errorf("e2e merged compile missing fresh goal:\n%s", merged)
	}
	if strings.Count(merged, RECALL_NOTE) != 1 {
		t.Errorf("e2e merged compile RECALL_NOTE count = %d, want 1:\n%s", strings.Count(merged, RECALL_NOTE), merged)
	}
}
