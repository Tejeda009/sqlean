// Package linter pkg/linter/diagnostic.go
package linter

import "github.com/Tejeda009/sqlean/pkg/lexer"

type Severity string

const (
	SeverityError   Severity = "ERROR"
	SeverityWarning Severity = "WARNING"
	SeverityInfo    Severity = "INFO"
)

// TextEdit defines a substitution to be done for autofix
type TextEdit struct {
	StartOffset int    `json:"start_offset"`
	EndOffset   int    `json:"end_offset"`
	Replacement string `json:"replacement"`
}

// Diagnostic is structured to be serialazed directly in JSON for VS code / CI         .
type Diagnostic struct {
	RuleID   string         `json:"rule_id"`
	Message  string         `json:"message"`
	Severity Severity       `json:"severity"`
	File     string         `json:"file,omitempty"`
	Start    lexer.Position `json:"start"`
	End      lexer.Position `json:"end"`
	Fix      *TextEdit      `json:"fix,omitempty"` //if present, can be applied with --fix or thorugh editor
}
