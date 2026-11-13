package vcc

// BuildSectionsInput holds the normalized blocks and optional file operations
// for producing a SectionData (algo spec §3.3, src/core/build-sections.ts).
type BuildSectionsInput struct {
	Blocks  []NormalizedBlock
	FileOps FileOps
}

// buildSections produces SectionData from normalized blocks and file operations
// (algo spec §3.3, src/core/build-sections.ts).
func buildSections(input BuildSectionsInput) SectionData {
	blocks := input.Blocks
	sessionGoal := extractGoals(blocks)
	userPreferences := dedupPreferencesAgainstGoals(
		extractPreferences(blocks),
		sessionGoal,
	)
	return SectionData{
		SessionGoal:        sessionGoal,
		OutstandingContext: extractOutstandingContext(blocks),
		FilesAndChanges:    formatFileActivity(blocks, input.FileOps),
		Commits:            formatCommits(extractCommits(blocks)),
		UserPreferences:    userPreferences,
		BriefTranscript:    stringifyBrief(buildBriefSections(blocks)),
	}
}
