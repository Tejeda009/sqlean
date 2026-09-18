// Package linter pkg/linter/rule.go
package linter

import (
	"github.com/Tejeda009/sqlean/pkg/ast"
	"github.com/Tejeda009/sqlean/pkg/dialect"
	"github.com/Tejeda009/sqlean/pkg/lexer"
)

// RuleContext contains all the info gave to the linter during code analisy.
type RuleContext struct {
	File       string
	Content    string
	Tokens     []lexer.Token
	Dialect    dialect.Dialect
	Statements []ast.Statement
}

// Rule defines the contract for every linter's rule (style, best practice, security).    .
type Rule interface {
	ID() string
	Name() string
	Description() string
	Run(ctx *RuleContext) []Diagnostic
}
