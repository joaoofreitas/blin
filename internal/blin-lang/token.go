package lexer

// TokenType identifies the kind of token scanned.
type TokenType int

const (
	TokenError TokenType = iota
	TokenEOF
	TokenText    // Standard Markdown body text
	TokenTag     // #tag
	TokenProject // +project
	TokenDate    // =yyyy-mm-dd or =dd-mm-yyyy
)

type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}
