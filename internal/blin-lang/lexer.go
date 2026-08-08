// Reference: https://go.dev/talks/2011/lex.slide#1

package lexer

import (
	"unicode/utf8"
)

// stateFn represents the state of the lexer as a function that returns the next state.
type stateFn func(*Lexer) stateFn

type Lexer struct {
	input     string  // The string being scanned
	start     int     // Start position of the current token
	pos       int     // Current position in the input
	width     int     // Width of the last rune read from input
	line      int     // Current line number (1-based)
	col       int     // Current column number (1-based)
	startLine int     // Line number where the current token started
	startCol  int     // Column number where the current token started
	tokens    []Token // Scanned tokens accumulator
}

// New initializes a new Lexer for the given input string.
func New(input string) *Lexer {
	return &Lexer{
		input:     input,
		line:      1,
		col:       1,
		startLine: 1,
		startCol:  1,
		tokens:    make([]Token, 0),
	}
}

// Run starts the state machine and returns all emitted tokens.
func (l *Lexer) Run() []Token {
	for state := lexText; state != nil; {
		state = state(l)
	}
	return l.tokens
}

// emit passes a token of the specified type back to the token accumulator.
func (l *Lexer) emit(t TokenType) {
	if l.pos > l.start {
		// Don't store text
		if t != TokenText {
			l.tokens = append(l.tokens, Token{
				Type:   t,
				Value:  l.input[l.start:l.pos],
				Line:   l.startLine,
				Column: l.startCol,
			})
		}
		l.start = l.pos
		l.startLine = l.line
		l.startCol = l.col
	}
}

// next returns the next rune in the input.
func (l *Lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return -1 // EOF
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width

	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return r
}

// backup steps back one rune. Can only be called once per call to next.
func (l *Lexer) backup() {
	if l.width > 0 {
		l.pos -= l.width
		if l.input[l.pos] == '\n' {
			l.line--
		} else {
			l.col--
		}
		l.width = 0
	}
}

// peek returns but does not consume the next rune in the input.
func (l *Lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}
