package vcc

import (
	"regexp"
	"strings"
)

// ── Preferences ──

var prefPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bprefer(?:s|red|ring)?\s+\w`),
	regexp.MustCompile(`(?i)\bdon'?t want\b`),
	regexp.MustCompile(`(?i)\balways (?:use|do|run|prefer|keep|make|format|write|add|set|put|prefix|start|include|append)\b`),
	regexp.MustCompile(`(?i)\bnever (?:use|do|run|push|commit|write|ignore|add|set|put|remove|delete|include|deploy)\b`),
	regexp.MustCompile(`(?i)\bplease (?:use|avoid|keep|make|don'?t|do not|format|write)\b`),
	regexp.MustCompile(`(?i)\b(?:style|format|language|naming)\s*[:=]\s*\S`),
}

// extractPreferences extracts up to 10 user preferences from user blocks
// (algo spec §3.3, ported from src/core/extract/preferences.ts).
func extractPreferences(blocks []NormalizedBlock) []string {
	var prefs []string
	seen := make(map[string]bool)

	for _, b := range blocks {
		if b.Kind != BlockKindUser {
			continue
		}
		perBlock := 0
		for _, line := range nonEmptyLines(b.Text) {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || len(trimmed) < 5 {
				continue
			}
			if len(trimmed) > 200 {
				continue
			}
			if strings.HasSuffix(trimmed, "?") || strings.Contains(trimmed, "?...") {
				continue
			}
			matched := false
			for _, p := range prefPatterns {
				if p.MatchString(trimmed) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
			clipped := clip(trimmed, 200)
			key := strings.ToLower(clipped)
			if seen[key] {
				continue
			}
			seen[key] = true
			prefs = append(prefs, clipped)
			perBlock++
			if perBlock >= 1 {
				break
			}
		}
	}

	if len(prefs) > 10 {
		prefs = prefs[:10]
	}
	return prefs
}

// dedupPreferencesAgainstGoals removes preferences that duplicate goals
// (case-insensitive, trimmed) (algo spec §3.3).
func dedupPreferencesAgainstGoals(prefs, goals []string) []string {
	goalSet := make(map[string]bool)
	for _, g := range goals {
		goalSet[strings.ToLower(strings.TrimSpace(g))] = true
	}
	var result []string
	for _, p := range prefs {
		key := strings.ToLower(strings.TrimSpace(p))
		if !goalSet[key] {
			result = append(result, p)
		}
	}
	return result
}
