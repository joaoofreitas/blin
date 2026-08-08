package lexer

import (
	"unicode"
)

// symbolTokens maps a trigger rune to the token type it starts.
var symbolTokens = map[rune]TokenType{
	'#': TokenTag,
	'+': TokenProject,
	'=': TokenDate,
}

// lexText is the primary state. It scans plain text until it encounters a trigger symbol.
func lexText(l *Lexer) stateFn {
	for {
		r := l.next()
		if r == -1 {
			l.emit(TokenText)
			l.emit(TokenEOF)
			return nil // Returning nil stops the l.Run() loop
		}

		tokType, ok := symbolTokens[r]
		if ok {
			l.backup()
			l.emit(TokenText)
			return lexSymbol(tokType)
		}
	}
}

// lexSymbol scans a #tag, +project or =date: the trigger rune followed by
// content characters.
func lexSymbol(t TokenType) stateFn {
	return func(l *Lexer) stateFn {
		l.next()
		next := l.peek()
		if unicode.IsSpace(next) || next == -1 || isSymbolBoundary(next) {
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
	return r == '+' || r == '=' || r == '#'
}
