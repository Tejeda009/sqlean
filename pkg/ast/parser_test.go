// pkg/ast/parser_test.go
package ast_test

import (
	"testing"

	"github.com/Tejeda009/sqlean/pkg/ast"
	"github.com/Tejeda009/sqlean/pkg/dialect"
)

func TestParseSelect(t *testing.T) {
	sql := `SELECT id, name FROM users WHERE age >= 18 GROUP BY id HAVING count(id) > 1 ORDER BY name DESC LIMIT 10 OFFSET 5;`
	stmts, err := ast.ParseSQL(sql, dialect.GetDefault())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}

	sel, ok := stmts[0].(*ast.SelectStmt)
	if !ok {
		t.Fatalf("expected *ast.SelectStmt, got %T", stmts[0])
	}

	if len(sel.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(sel.Columns))
	}
	if sel.From == nil {
		t.Errorf("expected FROM clause")
	}
	if sel.Where == nil {
		t.Errorf("expected WHERE clause")
	}
	if len(sel.GroupBy) != 1 {
		t.Errorf("expected 1 GROUP BY expr, got %d", len(sel.GroupBy))
	}
	if sel.Having == nil {
		t.Errorf("expected HAVING clause")
	}
	if len(sel.OrderBy) != 1 {
		t.Errorf("expected 1 ORDER BY expr, got %d", len(sel.OrderBy))
	}
	if sel.Limit == nil {
		t.Errorf("expected LIMIT clause")
	}
	if sel.Offset == nil {
		t.Errorf("expected OFFSET clause")
	}
}

func TestParseInsertUpdateDelete(t *testing.T) {
	sql := `
		INSERT INTO users (name, email) VALUES ('Alice', 'alice@test.com');
		UPDATE users SET name = 'Bob' WHERE id = 1;
		DELETE FROM users WHERE id = 1;
	`
	stmts, err := ast.ParseSQL(sql, dialect.GetDefault())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(stmts))
	}

	ins, ok := stmts[0].(*ast.InsertStmt)
	if !ok || len(ins.Columns) != 2 || len(ins.Values) != 1 {
		t.Errorf("unexpected insert stmt: %+v", ins)
	}

	upd, ok := stmts[1].(*ast.UpdateStmt)
	if !ok || len(upd.Assignments) != 1 || upd.Where == nil {
		t.Errorf("unexpected update stmt: %+v", upd)
	}

	del, ok := stmts[2].(*ast.DeleteStmt)
	if !ok || del.Where == nil {
		t.Errorf("unexpected delete stmt: %+v", del)
	}
}

func TestInspectAst(t *testing.T) {
	sql := `SELECT * FROM users WHERE active = true;`
	stmts, err := ast.ParseSQL(sql, dialect.GetDefault())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasStar := false
	ast.Inspect(stmts[0], func(n ast.Node) bool {
		if _, ok := n.(*ast.StarExpr); ok {
			hasStar = true
		}
		return true
	})

	if !hasStar {
		t.Errorf("expected to find *ast.StarExpr in query")
	}
}
