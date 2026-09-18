// pkg/lexer/lexer_test.go
package lexer_test

import (
	"testing"

	"github.com/Tejeda009/sqlean/pkg/dialect"
	"github.com/Tejeda009/sqlean/pkg/lexer"
)

func TestLexerBasic(t *testing.T) {
	sql := `SELECT id, name FROM "users" WHERE age >= 18 AND status = 'active';`
	d := dialect.GetDefault()
	lex := lexer.New(sql, d)
	tokens := lex.TokenizeCodeOnly()

	expected := []struct {
		tokType lexer.TokenType
		literal string
	}{
		{lexer.TokenKeyword, "SELECT"},
		{lexer.TokenIdentifier, "id"},
		{lexer.TokenPunctuation, ","},
		{lexer.TokenIdentifier, "name"},
		{lexer.TokenKeyword, "FROM"},
		{lexer.TokenIdentifier, `"users"`},
		{lexer.TokenKeyword, "WHERE"},
		{lexer.TokenIdentifier, "age"},
		{lexer.TokenOperator, ">="},
		{lexer.TokenNumberLiteral, "18"},
		{lexer.TokenKeyword, "AND"},
		{lexer.TokenIdentifier, "status"},
		{lexer.TokenOperator, "="},
		{lexer.TokenStringLiteral, "'active'"},
		{lexer.TokenPunctuation, ";"},
		{lexer.TokenEOF, ""},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %+v", len(expected), len(tokens), tokens)
	}

	for i, exp := range expected {
		if tokens[i].Type != exp.tokType || tokens[i].Literal != exp.literal {
			t.Errorf("token %d: expected (%v, %q), got (%v, %q)",
				i, exp.tokType, exp.literal, tokens[i].Type, tokens[i].Literal)
		}
	}
}

func TestLexerTriviaAndComments(t *testing.T) {
	sql := "-- initial comment\nSELECT /* block */ 1;"
	lex := lexer.New(sql, dialect.GetDefault())
	tokens := lex.TokenizeAll()

	var types []lexer.TokenType
	for _, tok := range tokens {
		types = append(types, tok.Type)
	}

	expectedTypes := []lexer.TokenType{
		lexer.TokenComment,
		lexer.TokenWhitespace,
		lexer.TokenKeyword,
		lexer.TokenWhitespace,
		lexer.TokenComment,
		lexer.TokenWhitespace,
		lexer.TokenNumberLiteral,
		lexer.TokenPunctuation,
		lexer.TokenEOF,
	}

	if len(types) != len(expectedTypes) {
		t.Fatalf("expected %d tokens, got %d (%v)", len(expectedTypes), len(types), types)
	}
	for i := range types {
		if types[i] != expectedTypes[i] {
			t.Errorf("token %d: expected type %v, got %v", i, expectedTypes[i], types[i])
		}
	}
}

func TestLexerPostgresCustomTokens(t *testing.T) {
	pg, _ := dialect.Get("postgres")
	sql := `SELECT $1, val::text FROM t WHERE data ->> 'key' = 'x';`
	lex := lexer.New(sql, pg)
	tokens := lex.TokenizeCodeOnly()

	// check $1 is placeholder
	if tokens[1].Type != lexer.TokenPlaceholder || tokens[1].Literal != "$1" {
		t.Errorf("expected $1 placeholder, got %+v", tokens[1])
	}
	// check :: is operator
	if tokens[4].Type != lexer.TokenOperator || tokens[4].Literal != "::" {
		t.Errorf("expected :: operator, got %+v", tokens[4])
	}
	// check ->> is operator
	if tokens[10].Type != lexer.TokenOperator || tokens[10].Literal != "->>" {
		t.Errorf("expected ->> operator, got %+v", tokens[10])
	}
}

func TestLexerMySQLCustomTokens(t *testing.T) {
	my, _ := dialect.Get("mysql")
	sql := "SELECT `col` FROM `tbl` WHERE id = ?;\n# commento mysql\nSELECT @var;"
	lex := lexer.New(sql, my)
	tokens := lex.TokenizeCodeOnly()

	// check `col` is identifier
	if tokens[1].Type != lexer.TokenIdentifier || tokens[1].Literal != "`col`" {
		t.Errorf("expected `col`, got %+v", tokens[1])
	}
	// check ? is placeholder
	if tokens[7].Type != lexer.TokenPlaceholder || tokens[7].Literal != "?" {
		t.Errorf("expected ?, got %+v", tokens[7])
	}
	// check @var is identifier
	if tokens[10].Type != lexer.TokenIdentifier || tokens[10].Literal != "@var" {
		t.Errorf("expected @var, got %+v", tokens[10])
	}
}
