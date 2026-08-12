package internal

import "time"

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

type TimeTotal struct {
	Hours float64
	Last  time.Time
}
