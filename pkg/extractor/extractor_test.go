// pkg/extractor/extractor_test.go
package extractor_test

import (
	goast "go/ast"
	goparser "go/parser"
	gotoken "go/token"
	"strings"
	"testing"

	"github.com/Tejeda009/sqlean/pkg/extractor"
)

func TestExtractFromGoSource(t *testing.T) {
	src := `package main

import "database/sql"

func getUsers(db *sql.DB) {
	
	query := ` + "`" + `SELECT id, name FROM users WHERE active = true` + "`" + `
	_ = query

	
	// sql
	custom := "SELECT count(*) FROM orders"
	_ = custom
}
`
	sqls, err := extractor.ExtractFromGoSource("main.go", []byte(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sqls) != 2 {
		t.Fatalf("expected 2 extracted queries, got %d", len(sqls))
	}

	if !strings.Contains(sqls[0].Query, "SELECT id, name FROM users") {
		t.Errorf("query 1 unexpected: %q", sqls[0].Query)
	}
	if !sqls[0].IsRawString {
		t.Errorf("query 1 should be raw string")
	}

	if !strings.Contains(sqls[1].Query, "SELECT count(*)") {
		t.Errorf("query 2 unexpected: %q", sqls[1].Query)
	}
}

func TestRewriteGoSource(t *testing.T) {
	src := []byte(`package main

func query() string {
	return "select id from users where id=1"
}
`)

	sqls, err := extractor.ExtractFromGoSource("test.go", src)
	if err != nil || len(sqls) == 0 {
		t.Fatalf("failed to extract query: %v", err)
	}

	replacements := []extractor.Replacement{
		{
			StartOffset: sqls[0].StartOffset,
			EndOffset:   sqls[0].EndOffset,
			NewSQL:      "SELECT\n  id\nFROM\n  users\nWHERE\n  id = 1;",
			IsRawString: sqls[0].IsRawString,
		},
	}

	rewritten := extractor.RewriteGoSource(src, replacements)

	fset := gotoken.NewFileSet()
	_, parseErr := goparser.ParseFile(fset, "test.go", rewritten, goparser.ParseComments)
	if parseErr != nil {
		t.Fatalf("rewritten Go code is invalid: %v\nContent:\n%s", parseErr, string(rewritten))
	}

	if !strings.Contains(string(rewritten), "SELECT") || !strings.Contains(string(rewritten), "users") {
		t.Errorf("rewritten code does not contain formatted SQL:\n%s", string(rewritten))
	}

	astNode, _ := goparser.ParseFile(fset, "test.go", rewritten, 0)
	goast.Inspect(astNode, func(n goast.Node) bool {
		if lit, ok := n.(*goast.BasicLit); ok && lit.Kind == gotoken.STRING {
			if !strings.HasPrefix(lit.Value, "`") {
				t.Errorf("expected multiline string to be converted to backticks, got %s", lit.Value)
			}
		}
		return true
	})
}
