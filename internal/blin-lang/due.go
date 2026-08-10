package lexer

import (
	"fmt"
	"strings"
	"time"
)

// ParseDueDate parses a due-date token in the form "=due:YYYYMMDD".
func ParseDueDate(val string) (time.Time, error) {
	if !strings.HasPrefix(val, "=due:") {
		return time.Time{}, fmt.Errorf("not a due date: %s", val)
	}
	return ParseDate("=" + strings.TrimPrefix(val, "=due:"))
}
