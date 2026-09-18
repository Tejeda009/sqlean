// Package rules pkg/linter/rules/rules.go
package rules

import (
	"fmt"
	"strings"

	"github.com/Tejeda009/sqlean/pkg/ast"
	"github.com/Tejeda009/sqlean/pkg/lexer"
	"github.com/Tejeda009/sqlean/pkg/linter"
)

// AllRules returns the full slice of built-in standard linting rules.
func AllRules() []linter.Rule {
	return []linter.Rule{
		&RuleUppercaseKeywords{},
		&RuleNoSelectStar{},
		&RuleTrailingWhitespace{},
		&RuleOperatorSpacing{},
		&RuleCommaSpacing{},
		&RuleRequireWhereUpdateDelete{},
		&RuleSemicolonTermination{},
	}
}

// --- L002: No SELECT * ---

type RuleNoSelectStar struct{}

func (r *RuleNoSelectStar) ID() string   { return "L002" }
func (r *RuleNoSelectStar) Name() string { return "no-select-star" }
func (r *RuleNoSelectStar) Description() string {
	return "Avoid SELECT * in production queries; explicitly list required columns for performance and maintainability."
}

func (r *RuleNoSelectStar) Run(ctx *linter.RuleContext) []linter.Diagnostic {
	var diags []linter.Diagnostic

	for _, stmt := range ctx.Statements {
		ast.Inspect(stmt, func(n ast.Node) bool {
			if star, ok := n.(*ast.StarExpr); ok {
				msg := "Use of SELECT * is discouraged in production. Explicitly specify columns."
				if star.Table != nil {
					msg = fmt.Sprintf("Use of %s.* is discouraged in production. Explicitly specify columns.", star.Table.NameToken.Literal)
				}
				diags = append(diags, linter.Diagnostic{
					RuleID:   r.ID(),
					Message:  msg,
					Severity: linter.SeverityWarning,
					File:     ctx.File,
					Start:    star.Pos(),
					End:      star.End(),
				})
			}
			return true
		})
	}

	return diags
}

// --- L003: Trailing Whitespace ---

type RuleTrailingWhitespace struct{}

func (r *RuleTrailingWhitespace) ID() string   { return "L003" }
func (r *RuleTrailingWhitespace) Name() string { return "no-trailing-whitespace" }
func (r *RuleTrailingWhitespace) Description() string {
	return "Trailing whitespace at line endings should be removed."
}

func (r *RuleTrailingWhitespace) Run(ctx *linter.RuleContext) []linter.Diagnostic {
	var diags []linter.Diagnostic
	lines := strings.Split(ctx.Content, "\n")
	offset := 0

	for lineIdx, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")
		if len(trimmed) < len(line) {
			startOff := offset + len(trimmed)
			endOff := offset + len(line)
			diags = append(diags, linter.Diagnostic{
				RuleID:   r.ID(),
				Message:  "Line contains trailing whitespace.",
				Severity: linter.SeverityInfo,
				File:     ctx.File,
				Start:    lexer.Position{Offset: startOff, Line: lineIdx + 1, Column: len(trimmed) + 1},
				End:      lexer.Position{Offset: endOff, Line: lineIdx + 1, Column: len(line) + 1},
				Fix: &linter.TextEdit{
					StartOffset: startOff,
					EndOffset:   endOff,
					Replacement: "",
				},
			})
		}
		offset += len(line) + 1
	}

	return diags
}

// --- L004: Operator Spacing ---

type RuleOperatorSpacing struct{}

func (r *RuleOperatorSpacing) ID() string   { return "L004" }
func (r *RuleOperatorSpacing) Name() string { return "operator-spacing" }
func (r *RuleOperatorSpacing) Description() string {
	return "Binary operators (=, !=, <, >, <=, >=, +, -) should be surrounded by single spaces."
}

func (r *RuleOperatorSpacing) Run(ctx *linter.RuleContext) []linter.Diagnostic {
	var diags []linter.Diagnostic
	tokens := ctx.Tokens

	for i := 1; i < len(tokens)-1; i++ {
		tok := tokens[i]
		if tok.Type != lexer.TokenOperator {
			continue
		}
		if tok.Literal == "::" || tok.Literal == "." {
			continue
		}

		prev := tokens[i-1]
		next := tokens[i+1]

		missingLeft := prev.Type != lexer.TokenWhitespace
		missingRight := next.Type != lexer.TokenWhitespace

		if missingLeft || missingRight {
			rep := " " + tok.Literal + " "
			startOff := tok.Start.Offset
			endOff := tok.End.Offset

			diags = append(diags, linter.Diagnostic{
				RuleID:   r.ID(),
				Message:  fmt.Sprintf("Operator %q should be surrounded by single spaces.", tok.Literal),
				Severity: linter.SeverityWarning,
				File:     ctx.File,
				Start:    tok.Start,
				End:      tok.End,
				Fix: &linter.TextEdit{
					StartOffset: startOff,
					EndOffset:   endOff,
					Replacement: rep,
				},
			})
		}
	}

	return diags
}

// --- L005: Comma Spacing ---

type RuleCommaSpacing struct{}

func (r *RuleCommaSpacing) ID() string   { return "L005" }
func (r *RuleCommaSpacing) Name() string { return "comma-spacing" }
func (r *RuleCommaSpacing) Description() string {
	return "Commas must not be preceded by whitespace and should be followed by a space or newline."
}

func (r *RuleCommaSpacing) Run(ctx *linter.RuleContext) []linter.Diagnostic {
	var diags []linter.Diagnostic
	tokens := ctx.Tokens

	for i := range tokens {
		tok := tokens[i]
		if tok.Type != lexer.TokenPunctuation || tok.Literal != "," {
			continue
		}

		if i > 0 && tokens[i-1].Type == lexer.TokenWhitespace {
			spaceTok := tokens[i-1]
			diags = append(diags, linter.Diagnostic{
				RuleID:   r.ID(),
				Message:  "Unexpected whitespace before comma.",
				Severity: linter.SeverityWarning,
				File:     ctx.File,
				Start:    spaceTok.Start,
				End:      spaceTok.End,
				Fix: &linter.TextEdit{
					StartOffset: spaceTok.Start.Offset,
					EndOffset:   spaceTok.End.Offset,
					Replacement: "",
				},
			})
		}

		if i+1 < len(tokens) && tokens[i+1].Type != lexer.TokenWhitespace && tokens[i+1].Type != lexer.TokenEOF {
			diags = append(diags, linter.Diagnostic{
				RuleID:   r.ID(),
				Message:  "Comma should be followed by a space or newline.",
				Severity: linter.SeverityWarning,
				File:     ctx.File,
				Start:    tok.Start,
				End:      tok.End,
				Fix: &linter.TextEdit{
					StartOffset: tok.Start.Offset,
					EndOffset:   tok.End.Offset,
					Replacement: ", ",
				},
			})
		}
	}

	return diags
}

// --- L006: Require WHERE on UPDATE / DELETE ---

type RuleRequireWhereUpdateDelete struct{}

func (r *RuleRequireWhereUpdateDelete) ID() string   { return "L006" }
func (r *RuleRequireWhereUpdateDelete) Name() string { return "require-where-update-delete" }
func (r *RuleRequireWhereUpdateDelete) Description() string {
	return "UPDATE and DELETE statements must contain a WHERE clause to prevent unintended full table modifications."
}

func (r *RuleRequireWhereUpdateDelete) Run(ctx *linter.RuleContext) []linter.Diagnostic {
	var diags []linter.Diagnostic

	for _, stmt := range ctx.Statements {
		switch s := stmt.(type) {
		case *ast.UpdateStmt:
			if s.Where == nil {
				diags = append(diags, linter.Diagnostic{
					RuleID:   r.ID(),
					Message:  "UPDATE statement is missing a WHERE clause.",
					Severity: linter.SeverityError,
					File:     ctx.File,
					Start:    s.Pos(),
					End:      s.End(),
				})
			}
		case *ast.DeleteStmt:
			if s.Where == nil {
				diags = append(diags, linter.Diagnostic{
					RuleID:   r.ID(),
					Message:  "DELETE statement is missing a WHERE clause.",
					Severity: linter.SeverityError,
					File:     ctx.File,
					Start:    s.Pos(),
					End:      s.End(),
				})
			}
		}
	}

	return diags
}

// --- L007: Semicolon Termination ---

type RuleSemicolonTermination struct{}

func (r *RuleSemicolonTermination) ID() string   { return "L007" }
func (r *RuleSemicolonTermination) Name() string { return "semicolon-termination" }
func (r *RuleSemicolonTermination) Description() string {
	return "SQL statements should terminate with a semicolon (;)."
}

func (r *RuleSemicolonTermination) Run(ctx *linter.RuleContext) []linter.Diagnostic {
	var diags []linter.Diagnostic
	tokens := ctx.Tokens

	var lastMeaningful *lexer.Token
	for i := len(tokens) - 1; i >= 0; i-- {
		t := &tokens[i]
		if t.Type != lexer.TokenEOF && t.Type != lexer.TokenWhitespace && t.Type != lexer.TokenComment {
			lastMeaningful = t
			break
		}
	}

	if lastMeaningful != nil && lastMeaningful.Literal != ";" {
		diags = append(diags, linter.Diagnostic{
			RuleID:   r.ID(),
			Message:  "Statement should terminate with a semicolon ';'.",
			Severity: linter.SeverityWarning,
			File:     ctx.File,
			Start:    lastMeaningful.End,
			End:      lastMeaningful.End,
			Fix: &linter.TextEdit{
				StartOffset: lastMeaningful.End.Offset,
				EndOffset:   lastMeaningful.End.Offset,
				Replacement: ";",
			},
		})
	}

	return diags
}
