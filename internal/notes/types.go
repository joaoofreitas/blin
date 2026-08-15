package notes

import "time"

// Note is a parsed markdown note with blin metadata.
type Note struct {
	Name        string
	Content     string
	Time        time.Time
	Due         time.Time
	TimeTracked map[string]TimeTotal
	Projects    []string
	Tags        []string
	FileRefs    []string
}

// TimeTotal aggregates tracked hours for a single ID.
type TimeTotal struct {
	Hours float64
	Last  time.Time
}

// TimeRow is one aggregated time-tracking total, sorted by Last descending.
type TimeRow struct {
	ID    string
	Total TimeTotal
}
