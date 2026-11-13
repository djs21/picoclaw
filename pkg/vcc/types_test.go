package vcc

import (
	"reflect"
	"testing"
)

func TestBlockKindConstants(t *testing.T) {
	if BlockKindUser != "user" {
		t.Errorf("BlockKindUser = %q, want %q", BlockKindUser, "user")
	}
	if BlockKindAssistant != "assistant" {
		t.Errorf("BlockKindAssistant = %q, want %q", BlockKindAssistant, "assistant")
	}
	if BlockKindToolCall != "tool_call" {
		t.Errorf("BlockKindToolCall = %q, want %q", BlockKindToolCall, "tool_call")
	}
	if BlockKindToolResult != "tool_result" {
		t.Errorf("BlockKindToolResult = %q, want %q", BlockKindToolResult, "tool_result")
	}
	if BlockKindBash != "bash" {
		t.Errorf("BlockKindBash = %q, want %q", BlockKindBash, "bash")
	}
}

func TestNormalizedBlockTags(t *testing.T) {
	b := NormalizedBlock{}
	typ := reflect.TypeOf(b)

	kind, _ := typ.FieldByName("Kind")
	if kind.Tag.Get("json") != "kind" {
		t.Errorf("NormalizedBlock.Kind json tag = %q, want %q", kind.Tag.Get("json"), "kind")
	}
	text, _ := typ.FieldByName("Text")
	if text.Tag.Get("json") != "text,omitempty" {
		t.Errorf("NormalizedBlock.Text json tag = %q, want %q", text.Tag.Get("json"), "text,omitempty")
	}
	sourceIndex, _ := typ.FieldByName("SourceIndex")
	if sourceIndex.Tag.Get("json") != "sourceIndex,omitempty" {
		t.Errorf("NormalizedBlock.SourceIndex json tag = %q, want %q", sourceIndex.Tag.Get("json"), "sourceIndex,omitempty")
	}
	name, _ := typ.FieldByName("Name")
	if name.Tag.Get("json") != "name,omitempty" {
		t.Errorf("NormalizedBlock.Name json tag = %q, want %q", name.Tag.Get("json"), "name,omitempty")
	}
}

func TestSectionDataTags(t *testing.T) {
	s := SectionData{}
	typ := reflect.TypeOf(s)

	fields := map[string]string{
		"SessionGoal":        "sessionGoal",
		"OutstandingContext": "outstandingContext",
		"FilesAndChanges":    "filesAndChanges",
		"Commits":            "commits",
		"UserPreferences":    "userPreferences",
		"BriefTranscript":    "briefTranscript",
	}
	for field, want := range fields {
		f, ok := typ.FieldByName(field)
		if !ok {
			t.Errorf("SectionData has no field %q", field)
			continue
		}
		if got := f.Tag.Get("json"); got != want {
			t.Errorf("SectionData.%s json tag = %q, want %q", field, got, want)
		}
	}
}
