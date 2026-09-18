// Package lexer pkg/lexer/token.go
package lexer

import "fmt"

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenWhitespace
	TokenComment
	TokenIdentifier
	TokenKeyword
	TokenStringLiteral
	TokenNumberLiteral
	TokenOperator
	TokenPunctuation
	TokenPlaceholder // Handles $1, ?, :name, @var
)

func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenWhitespace:
		return "WHITESPACE"
	case TokenComment:
		return "COMMENT"
	case TokenIdentifier:
		return "IDENTIFIER"
	case TokenKeyword:
		return "KEYWORD"
	case TokenStringLiteral:
		return "STRING"
	case TokenNumberLiteral:
		return "NUMBER"
	case TokenOperator:
		return "OPERATOR"
	case TokenPunctuation:
		return "PUNCTUATION"
	case TokenPlaceholder:
		return "PLACEHOLDER"
	default:
		return "UNKNOWN"
	}
}

// Position represents exact source code position (0-based byte offset, 1-based line/col).
type Position struct {
	Offset int `json:"offset"` // 0-based byte offset from start of input
	Line   int `json:"line"`   // 1-based
	Column int `json:"column"` // 1-based
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

type Token struct {
	Type    TokenType `json:"type"`
	Literal string    `json:"literal"`
	Start   Position  `json:"start"`
	End     Position  `json:"end"`
}

func (t Token) String() string {
	return fmt.Sprintf("<%s %q @ %s-%s>", t.Type, t.Literal, t.Start, t.End)
}
