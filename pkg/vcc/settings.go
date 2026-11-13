package vcc

// DEFAULT_SETTINGS is the MVP default settings scaffold (decision D11).
var DEFAULT_SETTINGS = Settings{
	SmartKeepTail: true,
}

// loadSettings returns the default settings. MVP scaffold: no persistent
// config source is wired yet, so it returns DEFAULT_SETTINGS.
func loadSettings() Settings {
	return DEFAULT_SETTINGS
}

// scaffoldSettings merges defaults into existing without clobbering set fields.
// Zero-valued fields are filled from DEFAULT_SETTINGS; explicitly set fields are
// preserved (never clobbered). Note: boolean fields cannot distinguish "unset
// false" from "explicitly false", so SmartKeepTail defaults to true when unset.
func scaffoldSettings(existing Settings) Settings {
	out := existing
	if !out.SmartKeepTail {
		out.SmartKeepTail = DEFAULT_SETTINGS.SmartKeepTail
	}
	// OverrideDefaultCompaction, ContinueAfterThresholdCompact default false;
	// ExcludeTools defaults nil. All are zero values, so nothing to fill.
	return out
}
