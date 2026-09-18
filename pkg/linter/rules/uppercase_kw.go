// Package rules pkg/linter/rules/uppercase_kw.go
package rules

import (
	"fmt"
	"strings"

	"github.com/Tejeda009/sqlean/pkg/lexer"
	"github.com/Tejeda009/sqlean/pkg/linter"
)

// RuleUppercaseKeywords (L001) enforces uppercase SQL keywords.
type RuleUppercaseKeywords struct{}

func (r *RuleUppercaseKeywords) ID() string {
	return "L001"
}

func (r *RuleUppercaseKeywords) Name() string {
	return "uppercase-keywords"
}

func (r *RuleUppercaseKeywords) Description() string {
	return "Reserved SQL keywords must be uppercase (e.g. SELECT instead of select)."
}

func (r *RuleUppercaseKeywords) Run(ctx *linter.RuleContext) []linter.Diagnostic {
	var diagnostics []linter.Diagnostic

	for _, tok := range ctx.Tokens {
		if tok.Type == lexer.TokenKeyword {
			upper := strings.ToUpper(tok.Literal)
			if tok.Literal != upper {
				diagnostics = append(diagnostics, linter.Diagnostic{
					RuleID:   r.ID(),
					Message:  fmt.Sprintf("Keyword %q should be uppercase (%q)", tok.Literal, upper),
					Severity: linter.SeverityWarning,
					File:     ctx.File,
					Start:    tok.Start,
					End:      tok.End,
					Fix: &linter.TextEdit{
						StartOffset: tok.Start.Offset,
						EndOffset:   tok.End.Offset,
						Replacement: upper,
					},
				})
			}
		}
	}

	return diagnostics
}
