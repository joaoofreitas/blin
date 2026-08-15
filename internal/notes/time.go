package notes

import (
	"fmt"
	"sort"
	"strings"
)

// AggregateTimeTotals merges time-tracking entries across notes.
// Rows are sorted by most recent tracking date, then ID.
func AggregateTimeTotals(ns []Note) []TimeRow {
	totals := make(map[string]TimeTotal)
	for _, note := range ns {
		for id, total := range note.TimeTracked {
			aggregate := totals[id]
			aggregate.Hours += total.Hours
			if total.Last.After(aggregate.Last) {
				aggregate.Last = total.Last
			}
			totals[id] = aggregate
		}
	}

	ids := make([]string, 0, len(totals))
	for id := range totals {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if totals[ids[i]].Last.Equal(totals[ids[j]].Last) {
			return ids[i] < ids[j]
		}
		return totals[ids[i]].Last.After(totals[ids[j]].Last)
	})

	rows := make([]TimeRow, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, TimeRow{ID: id, Total: totals[id]})
	}
	return rows
}

// FormatTimeTracking renders a compact summary of tracked IDs and hours.
func FormatTimeTracking(entries map[string]TimeTotal) string {
	if len(entries) == 0 {
		return ""
	}
	ids := make([]string, 0, len(entries))
	for id := range entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%s %.2gh", id, entries[id].Hours))
	}
	return strings.Join(parts, ", ")
}
