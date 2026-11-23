package vcc

import (
	"strings"
)

// ── File activity ──

// fileActivity holds categorized file paths.
type fileActivity struct {
	read     map[string]bool
	modified map[string]bool
	created  map[string]bool
}

// extractFiles returns all unique file paths observed from blocks tool calls
// (algo spec §3.3, ported from src/core/extract/files.ts).
func extractFiles(blocks []NormalizedBlock, fileOps FileOps) []string {
	_ = fileOps // FileOps is write-only; paths are observed from tool calls in blocks

	act := &fileActivity{
		read:     make(map[string]bool),
		modified: make(map[string]bool),
		created:  make(map[string]bool),
	}

	readTools := map[string]bool{"read": true, "read_file": true, "view": true}
	writeTools := map[string]bool{"edit": true, "write": true, "edit_file": true, "write_file": true, "multiedit": true, "quick_edit": true, "target_edit": true, "apply_patch": true}
	createTools := map[string]bool{"write": true, "write_file": true}

	for _, b := range blocks {
		if b.Kind != BlockKindToolCall {
			continue
		}
		p := extractPath(b.Args)
		if p == "" {
			continue
		}
		name := strings.ToLower(b.Name)
		if readTools[name] {
			act.read[p] = true
		}
		if writeTools[name] {
			act.modified[p] = true
		}
		if createTools[name] {
			act.created[p] = true
		}
	}

	// Trim common directory prefix.
	all := mapKeys(act.read, act.modified, act.created)
	prefix := longestCommonDirPrefix(all)
	if prefix != "" {
		for p := range act.read {
			if strings.HasPrefix(p, prefix) {
				delete(act.read, p)
				act.read[strings.TrimPrefix(p, prefix)] = true
			}
		}
		for p := range act.modified {
			if strings.HasPrefix(p, prefix) {
				delete(act.modified, p)
				act.modified[strings.TrimPrefix(p, prefix)] = true
			}
		}
		for p := range act.created {
			if strings.HasPrefix(p, prefix) {
				delete(act.created, p)
				act.created[strings.TrimPrefix(p, prefix)] = true
			}
		}
	}

	// Combine into a flat deduplicated list.
	result := make([]string, 0, len(act.read)+len(act.modified)+len(act.created))
	seen := make(map[string]bool)
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}
	for p := range act.read {
		add(p)
	}
	for p := range act.modified {
		add(p)
	}
	for p := range act.created {
		add(p)
	}
	return result
}

func mapKeys(maps ...map[string]bool) []string {
	all := make([]string, 0)
	for _, m := range maps {
		for k := range m {
			all = append(all, k)
		}
	}
	return all
}

// formatFileActivity produces formatted category lines: Modified, Created, Read.
// Modified wins over Created (a file that existed before is not listed as Created).
// Each category is capped at 10 entries with a "(+N more)" suffix (algo spec §3.3).
func formatFileActivity(blocks []NormalizedBlock, fileOps FileOps) []string {
	_ = fileOps // FileOps is write-only; paths are observed from tool calls in blocks

	readSet := make(map[string]bool)
	modifiedSet := make(map[string]bool)
	createdSet := make(map[string]bool)

	readTools := map[string]bool{"read": true, "read_file": true, "view": true}
	writeTools := map[string]bool{"edit": true, "write": true, "edit_file": true, "write_file": true, "multiedit": true, "quick_edit": true, "target_edit": true, "apply_patch": true}
	createTools := map[string]bool{"write": true, "write_file": true}

	for _, b := range blocks {
		if b.Kind != BlockKindToolCall {
			continue
		}
		p := extractPath(b.Args)
		if p == "" {
			continue
		}
		name := strings.ToLower(b.Name)
		if readTools[name] {
			readSet[p] = true
		}
		if writeTools[name] {
			modifiedSet[p] = true
		}
		if createTools[name] {
			createdSet[p] = true
		}
	}

	// Trim common directory prefix.
	all := mapKeys(readSet, modifiedSet, createdSet)
	prefix := longestCommonDirPrefix(all)
	if prefix != "" {
		trim := func(set map[string]bool) {
			for p := range set {
				if strings.HasPrefix(p, prefix) {
					delete(set, p)
					set[strings.TrimPrefix(p, prefix)] = true
				}
			}
		}
		trim(readSet)
		trim(modifiedSet)
		trim(createdSet)
	}

	// Modified wins over Created: drop Created paths already present as Modified.
	for p := range modifiedSet {
		delete(createdSet, p)
	}

	var lines []string
	catLimit := 10
	if len(modifiedSet) > 0 {
		arr := sortedKeys(modifiedSet)
		if len(arr) <= catLimit {
			lines = append(lines, "Modified: "+strings.Join(arr, ", "))
		} else {
			lines = append(lines, "Modified: "+strings.Join(arr[:catLimit], ", ")+" (+"+itoa(len(arr)-catLimit)+" more)")
		}
	}
	if len(createdSet) > 0 {
		arr := sortedKeys(createdSet)
		if len(arr) <= catLimit {
			lines = append(lines, "Created: "+strings.Join(arr, ", "))
		} else {
			lines = append(lines, "Created: "+strings.Join(arr[:catLimit], ", ")+" (+"+itoa(len(arr)-catLimit)+" more)")
		}
	}
	if len(readSet) > 0 {
		arr := sortedKeys(readSet)
		if len(arr) <= catLimit {
			lines = append(lines, "Read: "+strings.Join(arr, ", "))
		} else {
			lines = append(lines, "Read: "+strings.Join(arr[:catLimit], ", ")+" (+"+itoa(len(arr)-catLimit)+" more)")
		}
	}
	return lines
}

func sortedKeys(m map[string]bool) []string {
	arr := make([]string, 0, len(m))
	for k := range m {
		arr = append(arr, k)
	}
	for i := 1; i < len(arr); i++ {
		key := arr[i]
		j := i - 1
		for j >= 0 && arr[j] > key {
			arr[j+1] = arr[j]
			j--
		}
		arr[j+1] = key
	}
	return arr
}

func longestCommonDirPrefix(paths []string) string {
	var abs []string
	for _, p := range paths {
		if strings.HasPrefix(p, "/") {
			abs = append(abs, p)
		}
	}
	if len(abs) < 2 {
		return ""
	}
	split := make([][]string, len(abs))
	for i, p := range abs {
		split[i] = strings.Split(p, "/")
	}
	minLen := len(split[0])
	for _, s := range split {
		if len(s) < minLen {
			minLen = len(s)
		}
	}
	i := 0
	for i < minLen-1 {
		seg := split[0][i]
		same := true
		for _, s := range split {
			if s[i] != seg {
				same = false
				break
			}
		}
		if !same {
			break
		}
		i++
	}
	if i < 2 {
		return ""
	}
	return strings.Join(split[0][:i], "/") + "/"
}
