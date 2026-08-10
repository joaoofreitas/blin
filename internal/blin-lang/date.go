package lexer

import (
	"fmt"
	"strings"
	"time"
)

// ParseDate parses a date token in the form "=YYYYMMDD".
func ParseDate(val string) (time.Time, error) {
	val = strings.TrimPrefix(val, "=")

	val = strings.ReplaceAll(val, "/", "")
	val = strings.ReplaceAll(val, "-", "")

	if t, err := time.Parse("20060102", val); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid date format: %s", val)
}
