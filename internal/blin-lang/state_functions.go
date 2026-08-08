package lexer

import (
	"unicode"
)

// lexText is the primary state. It scans plain text until it encounters a trigger symbol.
func lexText(l *Lexer) stateFn {
	for {
		r := l.next()
		if r == -1 {
			l.emit(TokenText)
			l.emit(TokenEOF)
			return nil // Returning nil stops the l.Run() loop
		}

		if tokType, ok := symbolTokens[r]; ok {
			l.backup()
			l.emit(TokenText)
			return lexSymbol(tokType)
		}
	}
}

// symbolTokens maps a trigger rune to the token type it starts.
var symbolTokens = map[rune]TokenType{
	'#': TokenTag,
	'+': TokenProject,
	'=': TokenDate,
}

// lexSymbol scans a #tag, +project or =date: the trigger rune followed by
// non-space characters. If the trigger is immediately followed by a space
// (e.g. a Markdown heading "# Heading"), it's treated as plain text instead.
func lexSymbol(t TokenType) stateFn {
	return func(l *Lexer) stateFn {
		l.next() // consume trigger rune
		if unicode.IsSpace(l.peek()) || l.peek() == -1 {
			return lexText
		}
		for {
			r := l.next()
			if unicode.IsSpace(r) || r == -1 || isSymbolBoundary(r) {
				l.backup()
				l.emit(t)
				return lexText
			}
		}
	}
}

func isSymbolBoundary(r rune) bool {
	return r == '`' || r == '+' || r == '=' || r == '#'
}
