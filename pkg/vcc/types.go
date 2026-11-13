package vcc

// BlockKind classifies the kind of a normalized conversation block.
type BlockKind string

const (
	BlockKindUser      BlockKind = "user"
	BlockKindAssistant BlockKind = "assistant"
	BlockKindToolCall  BlockKind = "tool_call"
	BlockKindToolResult BlockKind = "tool_result"
	BlockKindBash      BlockKind = "bash"
)

// NormalizedBlock is the compiler's intermediate representation of a single
// conversation block produced by normalization (algo spec §2.1).
type NormalizedBlock struct {
	Kind        BlockKind              `json:"kind"`
	Text        string                 `json:"text,omitempty"`
	SourceIndex *int                   `json:"sourceIndex,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Args        map[string]interface{} `json:"args,omitempty"`
	Command     string                 `json:"command,omitempty"`
	Output      string                 `json:"output,omitempty"`
	ExitCode    *int                   `json:"exitCode,omitempty"`
}

// SectionData is the normalized intermediate representation rendered by the
// summary formatter (algo spec §2.2).
type SectionData struct {
	SessionGoal        []string `json:"sessionGoal"`
	OutstandingContext []string `json:"outstandingContext"`
	FilesAndChanges    []string `json:"filesAndChanges"`
	Commits            []string `json:"commits"`
	UserPreferences    []string `json:"userPreferences"`
	BriefTranscript    string   `json:"briefTranscript"`
}

// Settings is the single runtime settings struct produced by config.ParseSettings
// (decision D11, algo spec §4).
type Settings struct {
	OverrideDefaultCompaction       bool `json:"overrideDefaultCompaction"`
	SmartKeepTail                   bool `json:"smartKeepTail"`
	ContinueAfterThresholdCompact   bool `json:"continueAfterThresholdCompact"`
}

// FileOps records file activity observed during a session. It is the
// hook-provided, authoritative source for pre-compaction file activity
// (algo spec §3.3).
type FileOps interface {
	Read(path string)
	Modified(path string)
	Created(path string)
}
