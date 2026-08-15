package notes

import "strings"

// DisplayMetadata strips the leading '=' used by blin-lang tokens so
// tags/projects render as #tag and +project.
func DisplayMetadata(value string) string {
	return strings.TrimPrefix(value, "=")
}
