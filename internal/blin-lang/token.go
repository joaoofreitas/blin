package lexer

// TokenType identifies the kind of token scanned.
type TokenType int

const (
	TokenError TokenType = iota
	TokenEOF
	TokenText    // Standard Markdown body text
	TokenTag     // =#tag
	TokenProject // =+project
	TokenDate    // =YYYYMMDD
	TokenDue     // =due:YYYYMMDD
	TokenTime    // =tt:YYYYMMDD:(ID:hours)
	TokenBlin    // =blin:name
)

type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}
