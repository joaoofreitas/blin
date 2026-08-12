package lexer

import (
	"fmt"
	"strings"
)

func ParseBlin(val string) (string, error) {
	if !strings.HasPrefix(val, "=blin:") {
		return "", fmt.Errorf("not a blin entry: %s", val)
	}
	return strings.TrimPrefix(val, "=blin:"), nil
}
