package vcc

import (
	"encoding/json"
	"fmt"
)

// ParseSettings parses a ContextManagerConfig json.RawMessage into vcc.Settings
// defensively (algo spec §4). A nil or empty payload yields the zero Settings.
// Invalid JSON returns an error without panicking. Array fields that carry a
// non-array value are coerced to an empty slice rather than failing.
func ParseSettings(raw json.RawMessage) (Settings, error) {
	var s Settings
	if len(raw) == 0 {
		return s, nil
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return s, fmt.Errorf("parse vcc settings: %w", err)
	}

	if v, ok := obj["overrideDefaultCompaction"]; ok {
		if err := json.Unmarshal(v, &s.OverrideDefaultCompaction); err != nil {
			return s, fmt.Errorf("parse vcc settings overrideDefaultCompaction: %w", err)
		}
	}
	if v, ok := obj["smartKeepTail"]; ok {
		if err := json.Unmarshal(v, &s.SmartKeepTail); err != nil {
			return s, fmt.Errorf("parse vcc settings smartKeepTail: %w", err)
		}
	}
	if v, ok := obj["continueAfterThresholdCompact"]; ok {
		if err := json.Unmarshal(v, &s.ContinueAfterThresholdCompact); err != nil {
			return s, fmt.Errorf("parse vcc settings continueAfterThresholdCompact: %w", err)
		}
	}
	if v, ok := obj["excludeTools"]; ok {
		var arr []string
		if err := json.Unmarshal(v, &arr); err != nil {
			// Coerce bad arrays (e.g. a string where an array is expected) to [].
			arr = []string{}
		}
		if arr == nil {
			arr = []string{}
		}
		s.ExcludeTools = arr
	}

	return s, nil
}
