// pkg/dialect/dialect_test.go
package dialect_test

import (
	"testing"

	"github.com/Tejeda009/sqlean/pkg/dialect"
)

func TestDialectRegistry(t *testing.T) {
	d, err := dialect.Get("ansi")
	if err != nil {
		t.Fatalf("expected ansi dialect, got err: %v", err)
	}
	if d.Name() != "ansi" {
		t.Errorf("expected ansi, got %s", d.Name())
	}
	if !d.IsKeyword("SELECT") || !d.IsKeyword("select") {
		t.Errorf("SELECT should be a keyword")
	}
	if d.QuoteIdentifier("users") != `"users"` {
		t.Errorf("expected \"users\", got %s", d.QuoteIdentifier("users"))
	}

	pg, err := dialect.Get("postgres")
	if err != nil {
		t.Fatalf("expected postgres dialect, got err: %v", err)
	}
	if !pg.IsKeyword("RETURNING") {
		t.Errorf("RETURNING should be a postgres keyword")
	}

	tok, ok := pg.ScanCustomToken("$12 = test")
	if !ok || tok.Type != dialect.CustomTokenPlaceholder || tok.Literal != "$12" {
		t.Errorf("expected $12 placeholder, got %+v, ok=%v", tok, ok)
	}

	castTok, ok := pg.ScanCustomToken("::text")
	if !ok || castTok.Type != dialect.CustomTokenOperator || castTok.Literal != "::" {
		t.Errorf("expected :: operator, got %+v, ok=%v", castTok, ok)
	}

	my, err := dialect.Get("mysql")
	if err != nil {
		t.Fatalf("expected mysql dialect, got err: %v", err)
	}
	if my.QuoteIdentifier("table") != "`table`" {
		t.Errorf("expected `table`, got %s", my.QuoteIdentifier("table"))
	}

	varTok, ok := my.ScanCustomToken("@session_var := 1")
	if !ok || varTok.Type != dialect.CustomTokenIdentifier || varTok.Literal != "@session_var" {
		t.Errorf("expected @session_var, got %+v, ok=%v", varTok, ok)
	}

	qTok, ok := my.ScanCustomToken("? LIMIT 1")
	if !ok || qTok.Type != dialect.CustomTokenPlaceholder || qTok.Literal != "?" {
		t.Errorf("expected ? placeholder, got %+v, ok=%v", qTok, ok)
	}
}
