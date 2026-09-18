// Package ast pkg/ast/ast.go
package ast

import "github.com/Tejeda009/sqlean/pkg/lexer"

// Node is the base interface for all SQL AST nodes.
type Node interface {
	Pos() lexer.Position
	End() lexer.Position
}

// Statement represents a top-level SQL statement (SELECT, INSERT, UPDATE, DELETE, etc.).
type Statement interface {
	Node
	stmtNode()
}

// Expression represents an evaluable AST component (column, binary operation, literal, subquery).
type Expression interface {
	Node
	exprNode()
}

// --- Statements ---

type SelectStmt struct {
	SelectPos lexer.Position
	EndPos    lexer.Position
	Distinct  bool
	Columns   []Expression
	From      *FromClause
	Where     Expression
	GroupBy   []Expression
	Having    Expression
	OrderBy   []Expression
	Limit     Expression
	Offset    Expression
}

func (s *SelectStmt) Pos() lexer.Position { return s.SelectPos }
func (s *SelectStmt) End() lexer.Position { return s.EndPos }
func (s *SelectStmt) stmtNode()           {} // ok goland
func (s *SelectStmt) exprNode()           {} // this is ok

type InsertStmt struct {
	InsertPos lexer.Position
	EndPos    lexer.Position
	Table     Expression
	Columns   []Expression
	Values    [][]Expression
	Returning []Expression
}

func (s *InsertStmt) Pos() lexer.Position { return s.InsertPos }
func (s *InsertStmt) End() lexer.Position { return s.EndPos }
func (s *InsertStmt) stmtNode()           {} // bla bla

type UpdateStmt struct {
	UpdatePos   lexer.Position
	EndPos      lexer.Position
	Table       Expression
	Assignments []Assignment
	Where       Expression
	Returning   []Expression
}

func (s *UpdateStmt) Pos() lexer.Position { return s.UpdatePos }
func (s *UpdateStmt) End() lexer.Position { return s.EndPos }
func (s *UpdateStmt) stmtNode()           {} // yeah yeah

type DeleteStmt struct {
	DeletePos lexer.Position
	EndPos    lexer.Position
	Table     Expression
	Where     Expression
	Returning []Expression
}

func (s *DeleteStmt) Pos() lexer.Position { return s.DeletePos }
func (s *DeleteStmt) End() lexer.Position { return s.EndPos }
func (s *DeleteStmt) stmtNode()           {} // no problem

// RawStmt wraps unparsed or generic statements (e.g. DDL, DCL, or dialect extensions).
type RawStmt struct {
	StartPos lexer.Position
	EndPos   lexer.Position
	Tokens   []lexer.Token
}

func (r *RawStmt) Pos() lexer.Position { return r.StartPos }
func (r *RawStmt) End() lexer.Position { return r.EndPos }
func (r *RawStmt) stmtNode()           {} // trust me goland

// --- Expressions & Clauses ---

type Assignment struct {
	Column Expression
	Value  Expression
}

func (a Assignment) Pos() lexer.Position { return a.Column.Pos() }
func (a Assignment) End() lexer.Position { return a.Value.End() }

type FromClause struct {
	FromPos lexer.Position
	Table   Expression
	Joins   []JoinClause
}

func (f *FromClause) Pos() lexer.Position { return f.FromPos }
func (f *FromClause) End() lexer.Position {
	if len(f.Joins) > 0 {
		return f.Joins[len(f.Joins)-1].End()
	}
	if f.Table != nil {
		return f.Table.End()
	}
	return f.FromPos
}

type JoinClause struct {
	Type     string // INNER, LEFT, RIGHT, CROSS, FULL
	Table    Expression
	On       Expression
	Using    []Expression
	StartPos lexer.Position
	EndPos   lexer.Position
}

func (j JoinClause) Pos() lexer.Position { return j.StartPos }
func (j JoinClause) End() lexer.Position { return j.EndPos }

type Identifier struct {
	NameToken lexer.Token
}

func (i *Identifier) Pos() lexer.Position { return i.NameToken.Start }
func (i *Identifier) End() lexer.Position { return i.NameToken.End }
func (i *Identifier) exprNode()           {} // I don't want a yellow mark

type StarExpr struct {
	StarToken lexer.Token
	Table     *Identifier // optional table qualifier e.g. tbl.*
}

func (s *StarExpr) Pos() lexer.Position {
	if s.Table != nil {
		return s.Table.Pos()
	}
	return s.StarToken.Start
}
func (s *StarExpr) End() lexer.Position { return s.StarToken.End }
func (s *StarExpr) exprNode()           {} // ok, ok goland

type LiteralExpr struct {
	Token lexer.Token
}

func (l *LiteralExpr) Pos() lexer.Position { return l.Token.Start }
func (l *LiteralExpr) End() lexer.Position { return l.Token.End }
func (l *LiteralExpr) exprNode()           {} // perfect

type BinaryExpr struct {
	Left  Expression
	Op    lexer.Token
	Right Expression
}

func (b *BinaryExpr) Pos() lexer.Position { return b.Left.Pos() }
func (b *BinaryExpr) End() lexer.Position { return b.Right.End() }
func (b *BinaryExpr) exprNode()           {} // continue adding yellow marks

type UnaryExpr struct {
	Op   lexer.Token
	Expr Expression
}

func (u *UnaryExpr) Pos() lexer.Position { return u.Op.Start }
func (u *UnaryExpr) End() lexer.Position { return u.Expr.End() }
func (u *UnaryExpr) exprNode()           {} // nice

type FuncCallExpr struct {
	NameToken lexer.Token
	Args      []Expression
	EndPos    lexer.Position
}

func (f *FuncCallExpr) Pos() lexer.Position { return f.NameToken.Start }
func (f *FuncCallExpr) End() lexer.Position { return f.EndPos }
func (f *FuncCallExpr) exprNode()           {} // this is ok too

type ParenExpr struct {
	Expr   Expression
	Lparen lexer.Position
	Rparen lexer.Position
}

func (p *ParenExpr) Pos() lexer.Position { return p.Lparen }
func (p *ParenExpr) End() lexer.Position { return p.Rparen }
func (p *ParenExpr) exprNode()           {} // good

// --- Visitor Pattern & Inspect ---

type Visitor interface {
	Visit(node Node) (w Visitor)
}

func Walk(v Visitor, node Node) {
	if node == nil {
		return
	}
	v = v.Visit(node)
	if v == nil {
		return
	}

	switch n := node.(type) {
	case *SelectStmt:
		for _, col := range n.Columns {
			Walk(v, col)
		}
		if n.From != nil {
			if n.From.Table != nil {
				Walk(v, n.From.Table)
			}
			for _, j := range n.From.Joins {
				if j.Table != nil {
					Walk(v, j.Table)
				}
				if j.On != nil {
					Walk(v, j.On)
				}
				for _, u := range j.Using {
					Walk(v, u)
				}
			}
		}
		if n.Where != nil {
			Walk(v, n.Where)
		}
		for _, g := range n.GroupBy {
			Walk(v, g)
		}
		if n.Having != nil {
			Walk(v, n.Having)
		}
		for _, o := range n.OrderBy {
			Walk(v, o)
		}
		if n.Limit != nil {
			Walk(v, n.Limit)
		}
		if n.Offset != nil {
			Walk(v, n.Offset)
		}

	case *InsertStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
		for _, col := range n.Columns {
			Walk(v, col)
		}
		for _, row := range n.Values {
			for _, val := range row {
				Walk(v, val)
			}
		}
		for _, r := range n.Returning {
			Walk(v, r)
		}

	case *UpdateStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
		for _, a := range n.Assignments {
			if a.Column != nil {
				Walk(v, a.Column)
			}
			if a.Value != nil {
				Walk(v, a.Value)
			}
		}
		if n.Where != nil {
			Walk(v, n.Where)
		}
		for _, r := range n.Returning {
			Walk(v, r)
		}

	case *DeleteStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
		if n.Where != nil {
			Walk(v, n.Where)
		}
		for _, r := range n.Returning {
			Walk(v, r)
		}

	case *BinaryExpr:
		Walk(v, n.Left)
		Walk(v, n.Right)

	case *UnaryExpr:
		Walk(v, n.Expr)

	case *FuncCallExpr:
		for _, arg := range n.Args {
			Walk(v, arg)
		}

	case *ParenExpr:
		Walk(v, n.Expr)
	}
}

type inspector func(Node) bool

func (f inspector) Visit(node Node) Visitor {
	if f(node) {
		return f
	}
	return nil
}

// Inspect traverses an AST node depth-first, invoking f for each visited node.
func Inspect(node Node, f func(Node) bool) {
	Walk(inspector(f), node)
}
