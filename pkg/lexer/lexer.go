// Package lexer pkg/lexer/lexer.go
package lexer

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Tejeda009/sqlean/pkg/dialect"
)

type Lexer struct {
	input   string
	dialect dialect.Dialect
	pos     int
	line    int
	col     int
}

func New(input string, d dialect.Dialect) *Lexer {
	if d == nil {
		d = dialect.GetDefault()
	}
	return &Lexer{
		input:   input,
		dialect: d,
		pos:     0,
		line:    1,
		col:     1,
	}
}

// NextToken returns the next token from the input stream, preserving whitespace and comments (trivia).
func (l *Lexer) NextToken() Token {
	if l.pos >= len(l.input) {
		start := l.currentPos()
		return Token{Type: TokenEOF, Literal: "", Start: start, End: start}
	}

	start := l.currentPos()
	ch, width := utf8.DecodeRuneInString(l.input[l.pos:])

	// 1. Whitespace
	if unicode.IsSpace(ch) {
		return l.scanWhitespace()
	}

	// 2. Custom Dialect Tokens (placeholders, engine-specific operators, variables)
	if customTok, ok := l.dialect.ScanCustomToken(l.input[l.pos:]); ok {
		var tokType TokenType
		switch customTok.Type {
		case dialect.CustomTokenPlaceholder:
			tokType = TokenPlaceholder
		case dialect.CustomTokenOperator:
			tokType = TokenOperator
		case dialect.CustomTokenIdentifier:
			tokType = TokenIdentifier
		case dialect.CustomTokenLiteral:
			tokType = TokenStringLiteral
		case dialect.CustomTokenComment:
			tokType = TokenComment
		default:
			tokType = TokenIdentifier
		}
		l.advanceBytes(len(customTok.Literal))
		return Token{
			Type:    tokType,
			Literal: customTok.Literal,
			Start:   start,
			End:     l.currentPos(),
		}
	}

	// 3. Standard comments (-- or /* */)
	if ch == '-' && l.peek() == '-' {
		return l.scanLineComment()
	}
	if ch == '/' && l.peek() == '*' {
		return l.scanBlockComment()
	}

	// 4. String literals ('...')
	if ch == '\'' {
		return l.scanStringLiteral('\'')
	}

	// 5. Quoted Identifiers ("ident" or `ident`)
	if ch == '"' || ch == '`' {
		return l.scanQuotedIdentifier(ch)
	}

	// 6. Multi-character operators
	if op, ok := l.scanMultiCharOperator(); ok {
		l.advanceBytes(len(op))
		return Token{
			Type:    TokenOperator,
			Literal: op,
			Start:   start,
			End:     l.currentPos(),
		}
	}

	// 7. Numeric literals
	if unicode.IsDigit(ch) {
		lit := l.scanNumber()
		return Token{
			Type:    TokenNumberLiteral,
			Literal: lit,
			Start:   start,
			End:     l.currentPos(),
		}
	}

	// 8. Identifiers or Keywords
	if unicode.IsLetter(ch) || ch == '_' {
		lit := l.scanIdentOrKeyword()
		end := l.currentPos()
		tokType := TokenIdentifier
		if l.dialect.IsKeyword(lit) {
			tokType = TokenKeyword
		}
		return Token{
			Type:    tokType,
			Literal: lit,
			Start:   start,
			End:     end,
		}
	}

	// 9. Punctuation and single-character operators
	l.advanceBytes(width)
	lit := string(ch)
	tokType := TokenPunctuation
	if isSingleCharOperator(ch) {
		tokType = TokenOperator
	}

	return Token{
		Type:    tokType,
		Literal: lit,
		Start:   start,
		End:     l.currentPos(),
	}
}

func (l *Lexer) currentPos() Position {
	return Position{Offset: l.pos, Line: l.line, Column: l.col}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	_, width := utf8.DecodeRuneInString(l.input[l.pos:])
	if l.pos+width >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.pos+width:])
	return r
}

func (l *Lexer) advanceBytes(n int) {
	end := min(l.pos+n, len(l.input))
	for l.pos < end {
		r, width := utf8.DecodeRuneInString(l.input[l.pos:])
		l.pos += width
		if r == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
	}
}

func (l *Lexer) scanWhitespace() Token {
	start := l.currentPos()
	startPos := l.pos
	for l.pos < len(l.input) {
		r, w := utf8.DecodeRuneInString(l.input[l.pos:])
		if !unicode.IsSpace(r) {
			break
		}
		l.pos += w
		if r == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
	}
	return Token{
		Type:    TokenWhitespace,
		Literal: l.input[startPos:l.pos],
		Start:   start,
		End:     l.currentPos(),
	}
}

func (l *Lexer) scanLineComment() Token {
	start := l.currentPos()
	startPos := l.pos
	l.advanceBytes(2) // consume --
	for l.pos < len(l.input) {
		r, w := utf8.DecodeRuneInString(l.input[l.pos:])
		if r == '\n' {
			break
		}
		l.pos += w
		l.col++
	}
	return Token{
		Type:    TokenComment,
		Literal: l.input[startPos:l.pos],
		Start:   start,
		End:     l.currentPos(),
	}
}

func (l *Lexer) scanBlockComment() Token {
	start := l.currentPos()
	startPos := l.pos
	l.advanceBytes(2) // consume /*
	for l.pos < len(l.input)-1 {
		if l.input[l.pos] == '*' && l.input[l.pos+1] == '/' {
			l.advanceBytes(2)
			break
		}
		r, w := utf8.DecodeRuneInString(l.input[l.pos:])
		l.pos += w
		if r == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
	}
	return Token{
		Type:    TokenComment,
		Literal: l.input[startPos:l.pos],
		Start:   start,
		End:     l.currentPos(),
	}
}

func (l *Lexer) scanStringLiteral(quote rune) Token {
	start := l.currentPos()
	startPos := l.pos
	l.advanceBytes(1) // consume opening quote
	for l.pos < len(l.input) {
		r, w := utf8.DecodeRuneInString(l.input[l.pos:])
		if r == '\\' && l.pos+w < len(l.input) {
			// Escape sequence like \' or \\
			l.advanceBytes(w)
			nextR, nextW := utf8.DecodeRuneInString(l.input[l.pos:])
			_ = nextR
			l.advanceBytes(nextW)
			continue
		}
		if r == quote {
			l.advanceBytes(w)
			// Doubled quote escaping: 'O''Reilly'
			if l.pos < len(l.input) {
				nextR, nextW := utf8.DecodeRuneInString(l.input[l.pos:])
				if nextR == quote {
					l.advanceBytes(nextW)
					continue
				}
			}
			break
		}
		l.pos += w
		if r == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
	}
	return Token{
		Type:    TokenStringLiteral,
		Literal: l.input[startPos:l.pos],
		Start:   start,
		End:     l.currentPos(),
	}
}

func (l *Lexer) scanQuotedIdentifier(quote rune) Token {
	start := l.currentPos()
	startPos := l.pos
	l.advanceBytes(1) // consume opening quote
	for l.pos < len(l.input) {
		r, w := utf8.DecodeRuneInString(l.input[l.pos:])
		if r == quote {
			l.advanceBytes(w)
			// Doubled quote escaping: "" or ``
			if l.pos < len(l.input) {
				nextR, nextW := utf8.DecodeRuneInString(l.input[l.pos:])
				if nextR == quote {
					l.advanceBytes(nextW)
					continue
				}
			}
			break
		}
		l.pos += w
		if r == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
	}
	return Token{
		Type:    TokenIdentifier,
		Literal: l.input[startPos:l.pos],
		Start:   start,
		End:     l.currentPos(),
	}
}

func (l *Lexer) scanMultiCharOperator() (string, bool) {
	rem := l.input[l.pos:]
	ops := []string{"->>", "::", "<=", ">=", "<>", "!=", ":=", "||", "->"}
	for _, op := range ops {
		if strings.HasPrefix(rem, op) {
			return op, true
		}
	}
	return "", false
}

func isSingleCharOperator(ch rune) bool {
	switch ch {
	case '=', '<', '>', '+', '-', '*', '/', '%', '!', '~', '^', '&', '|':
		return true
	default:
		return false
	}
}

func (l *Lexer) scanIdentOrKeyword() string {
	start := l.pos
	for l.pos < len(l.input) {
		r, w := utf8.DecodeRuneInString(l.input[l.pos:])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			break
		}
		l.advanceBytes(w)
	}
	return l.input[start:l.pos]
}

func (l *Lexer) scanNumber() string {
	start := l.pos
	hasDot := false
	for l.pos < len(l.input) {
		r, w := utf8.DecodeRuneInString(l.input[l.pos:])
		if unicode.IsDigit(r) {
			l.advanceBytes(w)
		} else if r == '.' && !hasDot && l.pos+1 < len(l.input) && unicode.IsDigit(rune(l.input[l.pos+1])) {
			hasDot = true
			l.advanceBytes(w)
		} else {
			break
		}
	}
	return l.input[start:l.pos]
}

// TokenizeAll scans the entire input into a slice of tokens (including trivia).
func (l *Lexer) TokenizeAll() []Token {
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens
}

// TokenizeCodeOnly returns non-whitespace and non-comment tokens for parsing.
func (l *Lexer) TokenizeCodeOnly() []Token {
	var tokens []Token
	for {
		tok := l.NextToken()
		if tok.Type != TokenWhitespace && tok.Type != TokenComment {
			tokens = append(tokens, tok)
		}
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens
}
