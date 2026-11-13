package vcc

import (
	"sort"
	"strings"
)

// RECALL_NOTE is appended once at the end of a merged summary by the compile
// pipeline, never by mergePrevious itself.
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

// stripRecallNote removes a trailing RECALL_NOTE and its surrounding
// SECTION_SEPARATOR so the note is not restacked on merge.
func stripRecallNote(text string) string {
	idx := strings.LastIndex(text, RECALL_NOTE)
	if idx == -1 {
		return text
	}
	before := strings.TrimRight(text[:idx], "\n\r\t \f\v")
	before = strings.TrimSuffix(before, SECTION_SEPARATOR)
	return strings.TrimRight(before, "\n\r\t \f\v")
}

// mergePrevious merges a previous summary with a fresh one:
//   - sticky sections keep prev-then-fresh deduplicated lines, capped
//   - Files And Changes is merged per category with file-granular dedup
//   - Outstanding Context is volatile: fresh only
//   - brief transcripts are merged and capped at BRIEF_MAX_LINES
//   - empty prev yields fresh; empty fresh yields prev
//
// The RECALL_NOTE is NOT appended here; Compile is the single owner.
func mergePrevious(prev, fresh string) string {
	prev = stripRecallNote(prev)
	fresh = stripRecallNote(fresh)

	var headerParts []string
	for _, h := range headerOrder {
		prevLines := headerBlock(prev, h)
		freshLines := headerBlock(fresh, h)

		var merged []string
		switch h {
		case "Outstanding Context":
			merged = freshLines
		case "Files And Changes":
			merged = mergeFileLines(prevLines, freshLines)
		default:
			merged = dedupCap(prevLines, freshLines, capFor(h))
		}
		if len(merged) == 0 {
			continue
		}
		clean := make([]string, len(merged))
		for i, line := range merged {
			clean[i] = strings.TrimPrefix(line, "- ")
		}
		headerParts = append(headerParts, section(h, clean))
	}

	mergedBrief := mergeBriefTranscript(briefOf(prev), briefOf(fresh))
	if mergedBrief != "" {
		mergedBrief = capBrief(mergedBrief, BRIEF_MAX_LINES)
	}

	var parts []string
	if len(headerParts) > 0 {
		parts = append(parts, strings.Join(headerParts, "\n\n"))
	}
	if mergedBrief != "" {
		parts = append(parts, mergedBrief)
	}
	return strings.Join(parts, SECTION_SEPARATOR)
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

// briefOf returns the transcript after the final SECTION_SEPARATOR, or "".
func briefOf(text string) string {
	idx := strings.LastIndex(text, SECTION_SEPARATOR)
	if idx == -1 {
		return ""
	}
	return strings.TrimSpace(text[idx+len(SECTION_SEPARATOR):])
}

// mergeBriefTranscript joins previous and fresh brief transcripts; empty prev
// yields fresh and vice versa.
func mergeBriefTranscript(prev, fresh string) string {
	if prev == "" {
		return fresh
	}
	if fresh == "" {
		return prev
	}
	return prev + "\n\n" + fresh
}

// dedupCap deduplicates prev-then-fresh lines (order preserved) and keeps the
// last `cap` entries when over capacity.
func dedupCap(prev, fresh []string, cap int) []string {
	seen := make(map[string]bool)
	var out []string
	add := func(s string) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for _, s := range prev {
		add(s)
	}
	for _, s := range fresh {
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

type fileEntry struct {
	cat  string
	path string
}

// mergeFileLines merges "- Modified:", "- Created:", and "- Read:" lines by
// category. It splits comma-joined formatter output first, then unions paths
// per category (Modified wins over Created) and caps each category with a
// "(+N more)" suffix.
func mergeFileLines(prev, fresh []string) []string {
	catIndex := map[string]int{"Modified": 0, "Created": 1, "Read": 2}
	var combined []fileEntry
	seen := make(map[string]bool)
	add := func(cat, path string) {
		key := cat + "\x00" + path
		if seen[key] {
			return
		}
		seen[key] = true
		combined = append(combined, fileEntry{cat: cat, path: path})
	}

	for _, line := range append(prev, fresh...) {
		cat, paths := parseFileLine(line)
		for _, p := range paths {
			add(cat, p)
		}
	}

	modifiedSet := make(map[string]bool)
	for _, e := range combined {
		if e.cat == "Modified" {
			modifiedSet[e.path] = true
		}
	}

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
		if len(paths) == 0 {
			continue
		}
		sort.Strings(paths)
		if len(paths) <= fileCatCap {
			out = append(out, cat+": "+strings.Join(paths, ", "))
			continue
		}
		out = append(out, cat+": "+strings.Join(paths[:fileCatCap], ", ")+" (+"+itoa(len(paths)-fileCatCap)+" more)")
	}
	return out
}

// parseFileLine parses a line produced by headerBlock/formatter, e.g.
// "- Modified: a.go, b.go (+2 more)", into a category and individual paths.
func parseFileLine(line string) (string, []string) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "- ") {
		return "", nil
	}
	line = strings.TrimPrefix(line, "- ")
	if i := strings.Index(line, " ("); i != -1 {
		line = line[:i]
	}
	line = strings.TrimSpace(line)
	for _, cat := range []string{"Modified", "Created", "Read"} {
		prefix := cat + ":"
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		if rest == "" {
			return cat, nil
		}
		var paths []string
		for _, p := range strings.Split(rest, ",") {
			if p = strings.TrimSpace(p); p != "" {
				paths = append(paths, p)
			}
		}
		return cat, paths
	}
	return "", nil
}
