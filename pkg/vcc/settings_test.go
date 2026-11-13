package vcc

import (
	"reflect"
	"testing"
)

func TestDefaultSettings(t *testing.T) {
	if !DEFAULT_SETTINGS.SmartKeepTail {
		t.Error("DEFAULT_SETTINGS.SmartKeepTail = false, want true")
	}
	if DEFAULT_SETTINGS.OverrideDefaultCompaction {
		t.Error("DEFAULT_SETTINGS.OverrideDefaultCompaction = true, want false")
	}
	if DEFAULT_SETTINGS.ContinueAfterThresholdCompact {
		t.Error("DEFAULT_SETTINGS.ContinueAfterThresholdCompact = true, want false")
	}
	if DEFAULT_SETTINGS.ExcludeTools != nil {
		t.Errorf("DEFAULT_SETTINGS.ExcludeTools = %v, want nil", DEFAULT_SETTINGS.ExcludeTools)
	}
}

func TestLoadSettings(t *testing.T) {
	got := loadSettings()
	if !reflect.DeepEqual(got, DEFAULT_SETTINGS) {
		t.Errorf("loadSettings() = %+v, want %+v", got, DEFAULT_SETTINGS)
	}
}

func TestScaffoldSettingsAppliesDefaults(t *testing.T) {
	got := scaffoldSettings(Settings{})
	if !reflect.DeepEqual(got, DEFAULT_SETTINGS) {
		t.Errorf("scaffoldSettings(empty) = %+v, want %+v", got, DEFAULT_SETTINGS)
	}
}

func TestScaffoldSettingsNeverClobbersExisting(t *testing.T) {
	// Explicitly-set (non-default) values are preserved; unset fields take defaults.
	existing := Settings{
		OverrideDefaultCompaction: true,
		SmartKeepTail:             true,
		ExcludeTools:              []string{"WebSearch"},
	}
	got := scaffoldSettings(existing)
	if !got.OverrideDefaultCompaction {
		t.Error("scaffoldSettings clobbered OverrideDefaultCompaction")
	}
	if !got.SmartKeepTail {
		t.Error("scaffoldSettings clobbered SmartKeepTail")
	}
	if !reflect.DeepEqual(got.ExcludeTools, []string{"WebSearch"}) {
		t.Errorf("scaffoldSettings clobbered ExcludeTools = %v", got.ExcludeTools)
	}
	// ContinueAfterThresholdCompact was unset -> defaulted to false.
	if got.ContinueAfterThresholdCompact {
		t.Error("scaffoldSettings should default ContinueAfterThresholdCompact to false")
	}
}
