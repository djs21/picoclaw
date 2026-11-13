package vcc

// CompileInput contains the raw session messages and optional prior summary
// used by the deterministic VCC compiler (algo spec summarize.ts CompileInput).
type CompileInput struct {
	// Messages are raw session messages in the shape produced by the JSONL
	// backend before conversion ({role, content, ...}).
	Messages []map[string]interface{}

	// PreviousSummary is the prior VCC summary, if any. It is stripped of a
	// trailing RECALL_NOTE before merge to avoid restacking.
	PreviousSummary string

	// SourceIndices are optional session-global indices, parallel to Messages.
	// A missing entry renders no (#N) ref for that message.
	SourceIndices []int

	// FileOps is an optional hook-provided file activity tracker.
	FileOps FileOps
}

// Compile runs the deterministic VCC pipeline and returns a compact,
// merge-compatible summary ending with RECALL_NOTE (algo spec summarize.ts).
func Compile(input CompileInput) string {
	blocks := filterNoise(normalize(input.Messages, input.SourceIndices...))
	data := buildSections(BuildSectionsInput{
		Blocks:  blocks,
		FileOps: input.FileOps,
	})
	fresh := formatSummary(data)
	prev := stripRecallNote(input.PreviousSummary)

	var merged string
	if prev == "" {
		merged = fresh
	} else if fresh == "" {
		merged = prev
	} else {
		merged = mergePrevious(prev, fresh)
	}
	if merged == "" {
		return ""
	}
	return wrapLongLines(merged, TUI_SAFE_LINE_CHARS) + SECTION_SEPARATOR + RECALL_NOTE
}
