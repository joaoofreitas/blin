package lexer

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseTimeTrackingDate parses a time entry in the form
// "=tt:YYYYMMDD:(ID:hours)", where ID is a task identifier.
func ParseTimeTrackingDate(val string) (time.Time, string, float64, error) {
	if !strings.HasPrefix(val, "=tt:") {
		return time.Time{}, "", 0, fmt.Errorf("not a time-tracking entry: %s", val)
	}

	parts := strings.SplitN(strings.TrimPrefix(val, "=tt:"), ":", 3)
	if len(parts) != 3 {
		return time.Time{}, "", 0, fmt.Errorf("invalid time-tracking format: %s", val)
	}

	id := strings.Trim(strings.TrimSpace(parts[1]), "()")
	hoursText := strings.TrimSuffix(strings.Trim(strings.TrimSpace(parts[2]), "()"), "h")
	if id == "" {
		return time.Time{}, "", 0, fmt.Errorf("missing time-tracking ID: %s", val)
	}

	date, err := ParseDate("=" + parts[0])
	if err != nil {
		return time.Time{}, "", 0, fmt.Errorf("invalid time-tracking date: %w", err)
	}
	hours, err := strconv.ParseFloat(hoursText, 64)
	if err != nil {
		return time.Time{}, "", 0, fmt.Errorf("invalid time-tracking hours: %w", err)
	}
	return date, id, hours, nil
}
