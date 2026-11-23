package vcc

import (
	"regexp"
	"strings"
)

// SCOPE_CHANGE_RE matches signals that the user changed scope/task.
var SCOPE_CHANGE_RE = regexp.MustCompile(`(?i)\b(instead|actually|change of plan|forget that|new task|switch to|now I want|pivot|let'?s do|stop .* and)\b`)

// TASK_RE matches task-intent verbs at the start of a user block.
var TASK_RE = regexp.MustCompile(`(?i)\b(fix|implement|add|create|build|refactor|debug|investigate|update|remove|delete|migrate|deploy|test|write|set up)\b`)

// NOISE_SHORT_RE rejects very short non-goal responses.
var NOISE_SHORT_RE = regexp.MustCompile(`(?i)^(ok|yes|no|sure|yeah|yep|go|hi|hey|thx|thanks|y|n|k)\s*[.!?]*$`)

// NON_GOAL_RE rejects lines that are clearly not user goals.
var NON_GOAL_RE = regexp.MustCompile(`^\s*[\[│├└─╭╰]|^\s*(=[A-Z]+\(|func |const |let |var |import |export |class )|^(https?:|file:|/|\\)|\n|^\s*For each\b|\bin full\b[^\n]*\b(comments|issue|issues|PRs?|linked)\b`)

// TEMPLATE_SIGNAL_RE signals the start of a command template (stop collecting goals).
var TEMPLATE_SIGNAL_RE = regexp.MustCompile(`(?i)^\s*(For each\b|Do NOT implement\b|Analyze and propose\b|If Task\/context\b|Output:\s*$)`)

const maxGoalChars = 200
const leadingChars = 200

func truncateAtTemplate(lines []string) []string {
	for i, l := range lines {
		if TEMPLATE_SIGNAL_RE.MatchString(l) {
			return lines[:i]
		}
	}
	return lines
}

func stripLeadingBullet(line string) string {
	re := regexp.MustCompile(`^\s*(?:[-*+]|\d+\.)\s+`)
	return strings.TrimSpace(re.ReplaceAllString(line, ""))
}

func isSubstantiveGoal(text string) bool {
	t := strings.TrimSpace(text)
	if len(t) <= 5 {
		return false
	}
	if len(t) > maxGoalChars {
		return false
	}
	if NOISE_SHORT_RE.MatchString(t) {
		return false
	}
	if NON_GOAL_RE.MatchString(t) {
		return false
	}
	return true
}

func collapseSkillLines(lines []string) []string {
	var result []string
	seenSkills := make(map[string]bool)
	insideSkill := false
	skillTagRe := regexp.MustCompile(`^-?\s*<skill\s+name="([^"]+)"`)
	skillCloseRe := regexp.MustCompile(`^-?\s*</skill>`)
	for _, line := range lines {
		if skillTagRe.MatchString(line) {
			insideSkill = true
			m := skillTagRe.FindStringSubmatch(line)
			if len(m) > 1 && !seenSkills[m[1]] {
				seenSkills[m[1]] = true
				result = append(result, "[skill: "+m[1]+"]")
			}
			continue
		}
		if insideSkill {
			if skillCloseRe.MatchString(line) {
				insideSkill = false
			}
			continue
		}
		result = append(result, line)
	}
	return result
}

// extractGoals extracts up to 8 session goals from user blocks (algo spec §3.3).
func extractGoals(blocks []NormalizedBlock) []string {
	var goals []string
	var latestScopeChange []string

	for _, b := range blocks {
		if b.Kind != BlockKindUser {
			continue
		}
		rawLines := nonEmptyLines(b.Text)
		truncated := truncateAtTemplate(rawLines)
		lines := collapseSkillLines(truncated)
		lines = mapStripLeadingBullet(lines)
		lines = filterLenGT5(lines)
		lines = filterSubstantive(lines)
		if len(lines) == 0 {
			continue
		}
		if len(goals) == 0 {
			n := 6
			if len(lines) < n {
				n = len(lines)
			}
			goals = append(goals, lines[:n]...)
			continue
		}
		leading := b.Text
		if len(leading) > leadingChars {
			leading = leading[:leadingChars]
		}
		if SCOPE_CHANGE_RE.MatchString(leading) {
			n := 3
			if len(lines) < n {
				n = len(lines)
			}
			clipped := make([]string, n)
			for i := 0; i < n; i++ {
				clipped[i] = clip(lines[i], maxGoalChars)
			}
			latestScopeChange = clipped
		} else if TASK_RE.MatchString(leading) && len(lines) > 0 && len(lines[0]) > 15 {
			n := 2
			if len(lines) < n {
				n = len(lines)
			}
			clipped := make([]string, n)
			for i := 0; i < n; i++ {
				clipped[i] = clip(lines[i], maxGoalChars)
			}
			latestScopeChange = clipped
		}
	}

	if latestScopeChange != nil && len(latestScopeChange) > 0 {
		goals = append(goals, "[Scope change]")
		goals = append(goals, latestScopeChange...)
	}

	if len(goals) > 8 {
		goals = goals[:8]
	}
	return goals
}

func filterSubstantive(lines []string) []string {
	var out []string
	for _, l := range lines {
		if isSubstantiveGoal(l) {
			out = append(out, l)
		}
	}
	return out
}

func mapStripLeadingBullet(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = stripLeadingBullet(l)
	}
	return out
}

func filterLenGT5(lines []string) []string {
	var out []string
	for _, l := range lines {
		if len(l) > 5 {
			out = append(out, l)
		}
	}
	return out
}
