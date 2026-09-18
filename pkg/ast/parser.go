// Package ast pkg/ast/parser.go
package ast

import (
	"fmt"
	"strings"

	"github.com/Tejeda009/sqlean/pkg/dialect"
	"github.com/Tejeda009/sqlean/pkg/lexer"
)

type Parser struct {
	tokens  []lexer.Token
	cursor  int
	dialect dialect.Dialect
}

func NewParser(sql string, d dialect.Dialect) *Parser {
	if d == nil {
		d = dialect.GetDefault()
	}
	lex := lexer.New(sql, d)
	tokens := lex.TokenizeCodeOnly()
	return &Parser{
		tokens:  tokens,
		cursor:  0,
		dialect: d,
	}
}

// ParseSQL converts a raw SQL string into a slice of AST Statements.
func ParseSQL(sql string, d dialect.Dialect) ([]Statement, error) {
	p := NewParser(sql, d)
	return p.ParseAll()
}

func (p *Parser) ParseAll() ([]Statement, error) {
	var stmts []Statement

	for !p.isAtEnd() {
		// Skip extra semicolons
		if p.peek().Literal == ";" {
			p.advance()
			continue
		}

		tok := p.peek()
		upper := strings.ToUpper(tok.Literal)

		var stmt Statement
		var err error

		switch upper {
		case "SELECT":
			stmt, err = p.parseSelect()
		case "INSERT":
			stmt, err = p.parseInsert()
		case "UPDATE":
			stmt, err = p.parseUpdate()
		case "DELETE":
			stmt, err = p.parseDelete()
		default:
			stmt = p.parseRawStmt()
		}

		if err != nil {
			// Resilient fallback: convert failed statement to RawStmt
			raw := p.recoverToSemicolon(tok.Start)
			stmts = append(stmts, raw)
			continue
		}

		if stmt != nil {
			stmts = append(stmts, stmt)
		}

		// Consume trailing semicolon if present
		if !p.isAtEnd() && p.peek().Literal == ";" {
			p.advance()
		}
	}

	return stmts, nil
}

func (p *Parser) parseSelect() (*SelectStmt, error) {
	startTok := p.advance() // consume SELECT
	stmt := &SelectStmt{SelectPos: startTok.Start}

	if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "DISTINCT" {
		stmt.Distinct = true
		p.advance()
	}

	// 1. Target column projection
	for !p.isAtEnd() {
		upper := strings.ToUpper(p.peek().Literal)
		if upper == "FROM" || upper == "WHERE" || upper == "GROUP" || upper == "ORDER" || upper == "LIMIT" || p.peek().Literal == ";" {
			break
		}

		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		stmt.Columns = append(stmt.Columns, expr)

		// Optional alias with AS or positional
		if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "AS" {
			p.advance()
			if !p.isAtEnd() {
				p.advance() // consume alias
			}
		}

		if !p.isAtEnd() && p.peek().Literal == "," {
			p.advance() // consume comma
			continue
		}
		break
	}

	// 2. FROM Clause
	if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "FROM" {
		fromTok := p.advance()
		fromClause := &FromClause{FromPos: fromTok.Start}

		if !p.isAtEnd() {
			if p.peek().Literal == "(" {
				// Subquery in FROM
				tbl, err := p.parsePrimaryExpression()
				if err == nil {
					fromClause.Table = tbl
				}
			} else {
				fromClause.Table = p.parseTableRef()
			}
		}

		// Optional table alias
		if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "AS" {
			p.advance()
			if !p.isAtEnd() {
				p.advance()
			}
		}

		// JOIN clauses
		for !p.isAtEnd() {
			upper := strings.ToUpper(p.peek().Literal)
			isJoin := false
			joinType := ""

			if upper == "JOIN" {
				isJoin = true
				joinType = "INNER"
				p.advance()
			} else if upper == "INNER" || upper == "LEFT" || upper == "RIGHT" || upper == "FULL" || upper == "CROSS" {
				joinType = upper
				p.advance()
				if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "JOIN" {
					p.advance()
					isJoin = true
				}
			}

			if !isJoin {
				break
			}

			jc := JoinClause{Type: joinType, StartPos: p.previous().Start}
			if !p.isAtEnd() {
				jc.Table = p.parseTableRef()
			}

			// ON expression
			if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "ON" {
				p.advance()
				onExpr, _ := p.parseExpression()
				jc.On = onExpr
			}
			jc.EndPos = p.previous().End
			fromClause.Joins = append(fromClause.Joins, jc)
		}

		stmt.From = fromClause
	}

	// 3. WHERE Clause
	if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "WHERE" {
		p.advance()
		whereExpr, err := p.parseExpression()
		if err == nil {
			stmt.Where = whereExpr
		}
	}

	// 4. GROUP BY
	if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "GROUP" {
		p.advance()
		if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "BY" {
			p.advance()
			for !p.isAtEnd() {
				upper := strings.ToUpper(p.peek().Literal)
				if upper == "HAVING" || upper == "ORDER" || upper == "LIMIT" || p.peek().Literal == ";" {
					break
				}
				expr, err := p.parseExpression()
				if err == nil {
					stmt.GroupBy = append(stmt.GroupBy, expr)
				}
				if !p.isAtEnd() && p.peek().Literal == "," {
					p.advance()
					continue
				}
				break
			}
		}
	}

	// 5. HAVING
	if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "HAVING" {
		p.advance()
		havingExpr, err := p.parseExpression()
		if err == nil {
			stmt.Having = havingExpr
		}
	}

	// 6. ORDER BY
	if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "ORDER" {
		p.advance()
		if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "BY" {
			p.advance()
			for !p.isAtEnd() {
				upper := strings.ToUpper(p.peek().Literal)
				if upper == "LIMIT" || p.peek().Literal == ";" {
					break
				}
				expr, err := p.parseExpression()
				if err == nil {
					stmt.OrderBy = append(stmt.OrderBy, expr)
				}
				// ASC or DESC
				if !p.isAtEnd() {
					dir := strings.ToUpper(p.peek().Literal)
					if dir == "ASC" || dir == "DESC" {
						p.advance()
					}
				}
				if !p.isAtEnd() && p.peek().Literal == "," {
					p.advance()
					continue
				}
				break
			}
		}
	}

	// 7. LIMIT & OFFSET
	if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "LIMIT" {
		p.advance()
		limExpr, err := p.parseExpression()
		if err == nil {
			stmt.Limit = limExpr
		}
	}

	if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "OFFSET" {
		p.advance()
		offExpr, err := p.parseExpression()
		if err == nil {
			stmt.Offset = offExpr
		}
	}

	stmt.EndPos = p.previous().End
	return stmt, nil
}

func (p *Parser) parseInsert() (*InsertStmt, error) {
	insertTok := p.advance() // consume INSERT
	stmt := &InsertStmt{InsertPos: insertTok.Start}

	if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "INTO" {
		p.advance()
	}

	stmt.Table = p.parseTableRef()

	// Optional columns: (col1, col2)
	if !p.isAtEnd() && p.peek().Literal == "(" {
		p.advance() // consume (
		for !p.isAtEnd() && p.peek().Literal != ")" {
			col := p.parseTableRef()
			if col != nil {
				stmt.Columns = append(stmt.Columns, col)
			}
			if !p.isAtEnd() && p.peek().Literal == "," {
				p.advance()
				continue
			}
			break
		}
		if !p.isAtEnd() && p.peek().Literal == ")" {
			p.advance()
		}
	}

	// VALUES (...)
	if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "VALUES" {
		p.advance()
		for !p.isAtEnd() && p.peek().Literal == "(" {
			p.advance() // consume (
			var row []Expression
			for !p.isAtEnd() && p.peek().Literal != ")" {
				val, err := p.parseExpression()
				if err == nil {
					row = append(row, val)
				}
				if !p.isAtEnd() && p.peek().Literal == "," {
					p.advance()
					continue
				}
				break
			}
			if !p.isAtEnd() && p.peek().Literal == ")" {
				p.advance()
			}
			stmt.Values = append(stmt.Values, row)

			if !p.isAtEnd() && p.peek().Literal == "," {
				p.advance()
				continue
			}
			break
		}
	}

	stmt.EndPos = p.previous().End
	return stmt, nil
}

func (p *Parser) parseUpdate() (*UpdateStmt, error) {
	updTok := p.advance() // consume UPDATE
	stmt := &UpdateStmt{UpdatePos: updTok.Start}

	stmt.Table = p.parseTableRef()

	// SET col = val, ...
	if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "SET" {
		p.advance()
		for !p.isAtEnd() {
			if strings.ToUpper(p.peek().Literal) == "WHERE" || p.peek().Literal == ";" {
				break
			}
			col := p.parseTableRef()
			if col == nil {
				break
			}
			if !p.isAtEnd() && p.peek().Literal == "=" {
				p.advance()
			}
			val, err := p.parseExpression()
			if err != nil {
				break
			}
			stmt.Assignments = append(stmt.Assignments, Assignment{Column: col, Value: val})

			if !p.isAtEnd() && p.peek().Literal == "," {
				p.advance()
				continue
			}
			break
		}
	}

	// WHERE
	if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "WHERE" {
		p.advance()
		whereExpr, err := p.parseExpression()
		if err == nil {
			stmt.Where = whereExpr
		}
	}

	stmt.EndPos = p.previous().End
	return stmt, nil
}

func (p *Parser) parseDelete() (*DeleteStmt, error) {
	delTok := p.advance() // consume DELETE
	stmt := &DeleteStmt{DeletePos: delTok.Start}

	if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "FROM" {
		p.advance()
	}

	stmt.Table = p.parseTableRef()

	// WHERE
	if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "WHERE" {
		p.advance()
		whereExpr, err := p.parseExpression()
		if err == nil {
			stmt.Where = whereExpr
		}
	}

	stmt.EndPos = p.previous().End
	return stmt, nil
}

func (p *Parser) parseTableRef() Expression {
	if p.isAtEnd() {
		return nil
	}
	tok := p.advance()
	id := &Identifier{NameToken: tok}
	if !p.isAtEnd() && p.peek().Literal == "." {
		p.advance() // consume .
		if !p.isAtEnd() {
			col := p.advance()
			return &BinaryExpr{
				Left:  id,
				Op:    lexer.Token{Type: lexer.TokenPunctuation, Literal: "."},
				Right: &Identifier{NameToken: col},
			}
		}
	}
	return id
}

func (p *Parser) parseRawStmt() *RawStmt {
	startTok := p.peek()
	var tokens []lexer.Token

	for !p.isAtEnd() && p.peek().Literal != ";" {
		tokens = append(tokens, p.advance())
	}

	endPos := startTok.Start
	if len(tokens) > 0 {
		endPos = tokens[len(tokens)-1].End
	}

	return &RawStmt{
		StartPos: startTok.Start,
		EndPos:   endPos,
		Tokens:   tokens,
	}
}

func (p *Parser) recoverToSemicolon(start lexer.Position) *RawStmt {
	var tokens []lexer.Token
	for !p.isAtEnd() && p.peek().Literal != ";" {
		tokens = append(tokens, p.advance())
	}
	endPos := start
	if len(tokens) > 0 {
		endPos = tokens[len(tokens)-1].End
	}
	return &RawStmt{
		StartPos: start,
		EndPos:   endPos,
		Tokens:   tokens,
	}
}

func (p *Parser) parseExpression() (Expression, error) {
	return p.parseBinaryOrLogical(0)
}

func (p *Parser) parseBinaryOrLogical(minPrecedence int) (Expression, error) {
	left, err := p.parsePrimaryExpression()
	if err != nil {
		return nil, err
	}

	for !p.isAtEnd() {
		tok := p.peek()
		// Do not consume clause keywords or delimiters as binary operators
		upper := strings.ToUpper(tok.Literal)
		if upper == "FROM" || upper == "WHERE" || upper == "GROUP" || upper == "ORDER" ||
			upper == "HAVING" || upper == "LIMIT" || upper == "SET" || tok.Literal == ";" ||
			tok.Literal == "," || tok.Literal == ")" {
			break
		}

		prec := getOperatorPrecedence(tok)
		if prec < minPrecedence {
			break
		}

		op := p.advance()
		right, err := p.parseBinaryOrLogical(prec + 1)
		if err != nil {
			return left, nil
		}

		left = &BinaryExpr{
			Left:  left,
			Op:    op,
			Right: right,
		}
	}

	return left, nil
}

func getOperatorPrecedence(tok lexer.Token) int {
	upper := strings.ToUpper(tok.Literal)
	switch upper {
	case "OR":
		return 1
	case "AND":
		return 2
	case "=", "!=", "<>", "<", "<=", ">", ">=", "LIKE", "ILIKE", "IS", "IN":
		return 3
	case "+", "-":
		return 4
	case "*", "/", "%":
		return 5
	default:
		if tok.Type == lexer.TokenOperator {
			return 3
		}
		return -1
	}
}

func (p *Parser) parsePrimaryExpression() (Expression, error) {
	if p.isAtEnd() {
		return nil, fmt.Errorf("unexpected end of input")
	}

	tok := p.peek()

	// Star *
	if tok.Literal == "*" {
		p.advance()
		return &StarExpr{StarToken: tok}, nil
	}

	// Subquery or paren expression (...)
	if tok.Literal == "(" {
		lparen := p.advance().Start
		if !p.isAtEnd() && strings.ToUpper(p.peek().Literal) == "SELECT" {
			sub, err := p.parseSelect()
			if err != nil {
				return nil, err
			}
			rparen := p.previous().End
			if !p.isAtEnd() && p.peek().Literal == ")" {
				rparen = p.advance().End
			}
			return &ParenExpr{Expr: sub, Lparen: lparen, Rparen: rparen}, nil
		}

		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		rparen := p.previous().End
		if !p.isAtEnd() && p.peek().Literal == ")" {
			rparen = p.advance().End
		}
		return &ParenExpr{Expr: expr, Lparen: lparen, Rparen: rparen}, nil
	}

	// Numerics, Strings, Placeholders
	if tok.Type == lexer.TokenNumberLiteral || tok.Type == lexer.TokenStringLiteral || tok.Type == lexer.TokenPlaceholder {
		p.advance()
		return &LiteralExpr{Token: tok}, nil
	}

	// Functions or Identifiers
	if tok.Type == lexer.TokenIdentifier || tok.Type == lexer.TokenKeyword {
		p.advance()

		// Function call e.g. COUNT(id), NOW()
		if !p.isAtEnd() && p.peek().Literal == "(" {
			p.advance() // consume (
			var args []Expression
			for !p.isAtEnd() && p.peek().Literal != ")" {
				arg, err := p.parseExpression()
				if err == nil {
					args = append(args, arg)
				}
				if !p.isAtEnd() && p.peek().Literal == "," {
					p.advance()
					continue
				}
				break
			}
			endPos := p.previous().End
			if !p.isAtEnd() && p.peek().Literal == ")" {
				endPos = p.advance().End
			}
			return &FuncCallExpr{NameToken: tok, Args: args, EndPos: endPos}, nil
		}

		// Qualified identifier (e.g. tbl.* or tbl.col)
		if !p.isAtEnd() && p.peek().Literal == "." {
			p.advance() // consume .
			if !p.isAtEnd() {
				nextTok := p.advance()
				if nextTok.Literal == "*" {
					return &StarExpr{StarToken: nextTok, Table: &Identifier{NameToken: tok}}, nil
				}
				return &BinaryExpr{
					Left:  &Identifier{NameToken: tok},
					Op:    lexer.Token{Type: lexer.TokenPunctuation, Literal: "."},
					Right: &Identifier{NameToken: nextTok},
				}, nil
			}
		}

		return &Identifier{NameToken: tok}, nil
	}

	// Fallback token
	p.advance()
	return &LiteralExpr{Token: tok}, nil
}

func (p *Parser) peek() lexer.Token {
	if p.cursor >= len(p.tokens) {
		return lexer.Token{Type: lexer.TokenEOF}
	}
	return p.tokens[p.cursor]
}

func (p *Parser) previous() lexer.Token {
	if p.cursor == 0 {
		return lexer.Token{Type: lexer.TokenEOF}
	}
	return p.tokens[p.cursor-1]
}

func (p *Parser) advance() lexer.Token {
	if !p.isAtEnd() {
		p.cursor++
	}
	return p.previous()
}

func (p *Parser) isAtEnd() bool {
	return p.cursor >= len(p.tokens) || p.peek().Type == lexer.TokenEOF
}
