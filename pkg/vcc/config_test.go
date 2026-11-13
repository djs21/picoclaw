package vcc

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseSettingsNil(t *testing.T) {
	s, err := ParseSettings(nil)
	if err != nil {
		t.Fatalf("ParseSettings(nil) returned error: %v", err)
	}
	if !reflect.DeepEqual(s, Settings{}) {
		t.Errorf("ParseSettings(nil) = %+v, want zero Settings", s)
	}
}

func TestParseSettingsValid(t *testing.T) {
	raw := json.RawMessage(`{"overrideDefaultCompaction":true,"smartKeepTail":false,"continueAfterThresholdCompact":true,"excludeTools":["TodoWrite","WebSearch"]}`)
	s, err := ParseSettings(raw)
	if err != nil {
		t.Fatalf("ParseSettings(valid) returned error: %v", err)
	}
	if !s.OverrideDefaultCompaction {
		t.Error("OverrideDefaultCompaction = false, want true")
	}
	if s.SmartKeepTail {
		t.Error("SmartKeepTail = true, want false")
	}
	if !s.ContinueAfterThresholdCompact {
		t.Error("ContinueAfterThresholdCompact = false, want true")
	}
	want := []string{"TodoWrite", "WebSearch"}
	if !reflect.DeepEqual(s.ExcludeTools, want) {
		t.Errorf("ExcludeTools = %v, want %v", s.ExcludeTools, want)
	}
}

func TestParseSettingsInvalidJSON(t *testing.T) {
	raw := json.RawMessage(`{not valid json`)
	_, err := ParseSettings(raw)
	if err == nil {
		t.Error("ParseSettings(invalid JSON) expected error, got nil")
	}
}

func TestParseSettingsBadArrayCoerced(t *testing.T) {
	// excludeTools given a non-array value must coerce to [] rather than error.
	raw := json.RawMessage(`{"smartKeepTail":true,"excludeTools":"not-an-array"}`)
	s, err := ParseSettings(raw)
	if err != nil {
		t.Fatalf("ParseSettings(bad array) returned error: %v", err)
	}
	if !s.SmartKeepTail {
		t.Error("SmartKeepTail = false, want true")
	}
	if s.ExcludeTools == nil {
		t.Error("ExcludeTools = nil, want empty non-nil slice")
	}
	if len(s.ExcludeTools) != 0 {
		t.Errorf("ExcludeTools = %v, want empty slice", s.ExcludeTools)
	}
}
