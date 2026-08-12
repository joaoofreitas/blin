package lexer

import (
	"strings"
	"unicode"
)

// lexText scans plain text until it reaches an explicit metadata marker.
func lexText(l *Lexer) stateFn {
	for {
		r := l.next()
		if r == -1 {
			l.emit(TokenText)
			l.emit(TokenEOF)
			return nil // Returning nil stops the l.Run() loop
		}

		if r == '=' {
			l.backup()
			l.emit(TokenText)
			return lexSymbol
		}
	}
}

// lexSymbol recognizes =#tag, =+project, and =date-like metadata.
func lexSymbol(l *Lexer) stateFn {
	l.next() // Consume '='.
	next := l.peek()

	switch next {
	case '#':
		l.next()
		return lexMetadata(TokenTag)
	case '+':
		l.next()
		return lexMetadata(TokenProject)
	}

	if unicode.IsSpace(next) || next == -1 || next == '=' {
		return lexText
	}

	if strings.HasPrefix(l.input[l.start:], "=due:") {
		return lexMetadata(TokenDue)
	}
	if strings.HasPrefix(l.input[l.start:], "=tt:") {
		return lexMetadata(TokenTime)
	}

	if strings.HasPrefix(l.input[l.start:], "=blin:") {
		return lexMetadata(TokenBlin)
	}
	return lexMetadata(TokenDate)
}

func lexMetadata(t TokenType) stateFn {
	return func(l *Lexer) stateFn {
		for {
			r := l.next()
			if unicode.IsSpace(r) || r == -1 || r == '=' {
				l.backup()
				l.emit(t)
				return lexText
			}
		}
	}
}
