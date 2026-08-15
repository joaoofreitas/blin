package notes

import (
	"slices"
	"sort"
)

// Filter returns notes matching the optional tag and project filters.
// Empty filter strings are ignored.
func Filter(notes []Note, tag, project string) []Note {
	var filtered []Note
	for _, note := range notes {
		if tag != "" && !slices.Contains(note.Tags, tag) {
			continue
		}
		if project != "" && !slices.Contains(note.Projects, project) {
			continue
		}
		filtered = append(filtered, note)
	}
	return filtered
}

// WithDue returns notes that have a due date, soonest first.
func WithDue(notes []Note) []Note {
	var filtered []Note
	for _, note := range notes {
		if !note.Due.IsZero() {
			filtered = append(filtered, note)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Due.Before(filtered[j].Due)
	})
	return filtered
}

// WithTimeTracked returns notes that contain at least one time-tracking entry.
func WithTimeTracked(notes []Note) []Note {
	var filtered []Note
	for _, note := range notes {
		if len(note.TimeTracked) > 0 {
			filtered = append(filtered, note)
		}
	}
	return filtered
}

// Tags returns sorted unique tags across notes.
func Tags(notes []Note) []string {
	values := make(map[string]struct{})
	for _, note := range notes {
		for _, tag := range note.Tags {
			values[tag] = struct{}{}
		}
	}
	return sortedKeys(values)
}

// Projects returns sorted unique projects across notes.
func Projects(notes []Note) []string {
	values := make(map[string]struct{})
	for _, note := range notes {
		for _, project := range note.Projects {
			values[project] = struct{}{}
		}
	}
	return sortedKeys(values)
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
