// Package linter pkg/linter/engine.go
package linter

import (
	"sort"

	"github.com/Tejeda009/sqlean/pkg/ast"
	"github.com/Tejeda009/sqlean/pkg/dialect"
	"github.com/Tejeda009/sqlean/pkg/lexer"
)

type Engine struct {
	rules []Rule
}

func NewEngine() *Engine {
	return &Engine{rules: make([]Rule, 0)}
}

func (e *Engine) RegisterRule(r Rule) {
	e.rules = append(e.rules, r)
}

func (e *Engine) RegisterRules(rules ...Rule) {
	e.rules = append(e.rules, rules...)
}

func (e *Engine) Rules() []Rule {
	return e.rules
}

// Lint executes every rules registered on the sql codde provided  .
func (e *Engine) Lint(filename, content string, d dialect.Dialect) ([]Diagnostic, error) {
	if d == nil {
		d = dialect.GetDefault()
	}

	lex := lexer.New(content, d)
	tokens := lex.TokenizeAll()

	stmts, _ := ast.ParseSQL(content, d)

	ctx := &RuleContext{
		File:       filename,
		Content:    content,
		Tokens:     tokens,
		Dialect:    d,
		Statements: stmts,
	}

	var diagnostics []Diagnostic
	for _, rule := range e.rules {
		diags := rule.Run(ctx)
		diagnostics = append(diagnostics, diags...)
	}

	// Sort diagnostics for rows and column cresc
	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].File != diagnostics[j].File {
			return diagnostics[i].File < diagnostics[j].File
		}
		if diagnostics[i].Start.Line != diagnostics[j].Start.Line {
			return diagnostics[i].Start.Line < diagnostics[j].Start.Line
		}
		if diagnostics[i].Start.Column != diagnostics[j].Start.Column {
			return diagnostics[i].Start.Column < diagnostics[j].Start.Column
		}
		return diagnostics[i].RuleID < diagnostics[j].RuleID
	})

	return diagnostics, nil
}

// ApplyFixes applyes fixes auto suggested
func ApplyFixes(originalContent string, diagnostics []Diagnostic) string {
	var fixes []TextEdit
	for _, d := range diagnostics {
		if d.Fix != nil {
			fixes = append(fixes, *d.Fix)
		}
	}

	if len(fixes) == 0 {
		return originalContent
	}

	// Sort in desc order by StartOffset
	sort.Slice(fixes, func(i, j int) bool {
		if fixes[i].StartOffset != fixes[j].StartOffset {
			return fixes[i].StartOffset > fixes[j].StartOffset
		}
		return fixes[i].EndOffset > fixes[j].EndOffset
	})

	result := originalContent
	lastAppliedStart := len(result) + 1

	for _, fix := range fixes {
		// Prevents sovrappositions of contrasting edits
		if fix.EndOffset > lastAppliedStart {
			continue
		}
		if fix.StartOffset >= 0 && fix.EndOffset <= len(result) && fix.StartOffset <= fix.EndOffset {
			result = result[:fix.StartOffset] + fix.Replacement + result[fix.EndOffset:]
			lastAppliedStart = fix.StartOffset
		}
	}

	return result
}
