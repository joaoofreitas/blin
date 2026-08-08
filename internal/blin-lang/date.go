package lexer

import (
	"fmt"
	"strings"
	"time"
)

// ParseDate takes a TokenDate value (e.g. "=20260808" or "=2026/08/08") and parses it into a time.Time.
func ParseDate(val string) (time.Time, error) {
	val = strings.TrimPrefix(val, "=")
	
	// Remove slashes or hyphens to normalize to YYYYMMDD
	val = strings.ReplaceAll(val, "/", "")
	val = strings.ReplaceAll(val, "-", "")

	if t, err := time.Parse("20060102", val); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid date format: %s", val)
}
