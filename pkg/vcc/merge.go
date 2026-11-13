package vcc

import (
	"strings"
)

// RECALL_NOTE is appended once at the end of a merged summary to avoid
// compounding across compactions (algo spec §3.6; format.ts:49).
const RECALL_NOTE = "Use vcc_recall to search for prior work, decisions, and context from before this summary. Do not redo work already completed."

const (
	sessionGoalCap = 8
	commitsCap     = 8
	userPrefsCap   = 15
	fileCatCap     = 10
)

// headerOrder is the canonical section order (algo spec §3.6).
var headerOrder = []string{
	"Session Goal",
	"Files And Changes",
	"Commits",
	"Outstanding Context",
	"User Preferences",
}

// stripRecallNote removes a trailing RECALL_NOTE (and surrounding
// separators/whitespace) via lastIndexOf so it isn't restacked on merge
// (algo spec §3.6, summarize.ts:207).
func stripRecallNote(text string) string {
	idx := strings.LastIndex(text, RECALL_NOTE)
	if idx == -1 {
		return text
	}
	before := text[:idx]
	return strings.TrimRight(before, "\n\r\t \f\v")
}

// mergePrevious merges a previous summary with a fresh one (algo spec §3.6):
//
//	- sticky sections (Session Goal, Commits, User Preferences): line-level
//	  dedup of prev then fresh, capped per section
//	- Files And Changes: merged by category via mergeFileLines
//	- Outstanding Context is volatile: fresh only
//	- empty prev yields fresh; empty fresh yields prev
//	- RECALL_NOTE is appended once at the end
func mergePrevious(prev, fresh string) string {
	prev = stripRecallNote(prev)

	var parts []string
	for _, h := range headerOrder {
		pSec := headerBlock(prev, h)
		fSec := headerBlock(fresh, h)

		var merged []string
		switch h {
		case "Outstanding Context":
			// volatile: fresh only
			merged = fSec
		case "Files And Changes":
			merged = mergeFileLines(pSec, fSec)
		default:
			// sticky: line-level dedup prev+fresh, capped
			merged = dedupCap(pSec, fSec, capFor(h))
		}

		if len(merged) == 0 {
			continue
		}
		b := new(strings.Builder)
		b.WriteString("[" + h + "]")
		for _, line := range merged {
			b.WriteString("\n" + line)
		}
		parts = append(parts, b.String())
	}

	merged := strings.TrimSpace(strings.Join(parts, "\n\n"))
	if merged != "" {
		merged += "\n\n"
	}
	return merged + RECALL_NOTE
}

// headerBlock returns the content lines (including the "- " prefix) under the
// given [header] in text, stopping at the next header or end of text.
func headerBlock(text, header string) []string {
	token := "[" + header + "]"
	idx := strings.Index(text, token)
	if idx == -1 {
		return nil
	}
	rest := text[idx+len(token):]

	next := -1
	for _, h := range headerOrder {
		if h == header {
			continue
		}
		if j := strings.Index(rest, "\n["+h+"]"); j != -1 && (next == -1 || j < next) {
			next = j
		}
	}
	if next != -1 {
		rest = rest[:next]
	}

	var lines []string
	for _, raw := range strings.Split(rest, "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		if strings.Contains(line, "<skill") || strings.Contains(line, "</skill") {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

// dedupCap returns prev-then-fresh line-level deduplicated, keeping the last
// `cap` entries when over capacity.
func dedupCap(prev, fresh []string, cap int) []string {
	seen := make(map[string]bool)
	var out []string
	add := func(s string) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for _, s := range append(prev, fresh...) {
		add(s)
	}
	if cap > 0 && len(out) > cap {
		out = out[len(out)-cap:]
	}
	return out
}

// capFor returns the sticky-section line cap for header.
func capFor(header string) int {
	switch header {
	case "Session Goal":
		return sessionGoalCap
	case "Commits":
		return commitsCap
	case "User Preferences":
		return userPrefsCap
	}
	return 0
}

// mergeFileLines merges "- Modified:", "- Created:", "- Read:" lines by
// category: union paths per category, dedup (Modified wins over Created), cap
// each category at fileCatCap with a "(+N more)" suffix (algo spec §3.6).
func mergeFileLines(prev, fresh []string) []string {
	type entry struct {
		cat, path string
	}

	combined := []entry{}
	seen := make(map[string]bool)
	add := func(cat, path string) {
		key := cat + "\x00" + path
		if seen[key] {
			return
		}
		seen[key] = true
		combined = append(combined, entry{cat, path})
	}

	parse := func(line string) (cat, path string, ok bool) {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- ") {
			return "", "", false
		}
		line = strings.TrimPrefix(line, "- ")
		if i := strings.Index(line, " ("); i != -1 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		for _, cat := range []string{"Modified", "Created", "Read"} {
			prefix := cat + ":"
			if strings.HasPrefix(line, prefix) {
				path := strings.TrimSpace(strings.TrimPrefix(line, prefix))
				return cat, path, path != ""
			}
		}
		return "", "", false
	}

	for _, line := range append(prev, fresh...) {
		if cat, path, ok := parse(line); ok {
			add(cat, path)
		}
	}

	// Modified wins over Created: drop Created paths already present as Modified.
	modifiedSet := make(map[string]bool)
	for _, e := range combined {
		if e.cat == "Modified" {
			modifiedSet[e.path] = true
		}
	}

	catIndex := map[string]int{"Modified": 0, "Created": 1, "Read": 2}
	byCat := [3][]string{}
	for _, e := range combined {
		if e.cat == "Created" && modifiedSet[e.path] {
			continue
		}
		byCat[catIndex[e.cat]] = append(byCat[catIndex[e.cat]], e.path)
	}

	catOrder := []string{"Modified", "Created", "Read"}
	var out []string
	for _, cat := range catOrder {
		paths := byCat[catIndex[cat]]
		for i, p := range paths {
			if i >= fileCatCap {
				break
			}
			line := "- " + cat + ": " + p
			if i == fileCatCap-1 && len(paths) > fileCatCap {
				line += " (+" + itoa(len(paths)-fileCatCap) + " more)"
			}
			out = append(out, line)
		}
	}
	return out
}

// itoa is a minimal integer-to-string converter (avoids strconv import).
func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
