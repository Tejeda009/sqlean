// pkg/linter/linter_test.go
package linter_test

import (
	"strings"
	"testing"

	"github.com/Tejeda009/sqlean/pkg/dialect"
	"github.com/Tejeda009/sqlean/pkg/linter"
	"github.com/Tejeda009/sqlean/pkg/linter/rules"
)

func setupEngine() *linter.Engine {
	eng := linter.NewEngine()
	eng.RegisterRules(rules.AllRules()...)
	return eng
}

func TestLinterRules(t *testing.T) {
	eng := setupEngine()
	d := dialect.GetDefault()

	// 1. Test L001 & L007: lowercase keywords and missing semicolon
	sql1 := "select id from users"
	diags1, err := eng.Lint("test1.sql", sql1, d)
	if err != nil {
		t.Fatalf("lint error: %v", err)
	}

	hasL001 := false
	hasL007 := false
	for _, diag := range diags1 {
		if diag.RuleID == "L001" {
			hasL001 = true
		}
		if diag.RuleID == "L007" {
			hasL007 = true
		}
	}
	if !hasL001 {
		t.Errorf("expected L001 (uppercase-keywords) violation")
	}
	if !hasL007 {
		t.Errorf("expected L007 (semicolon-termination) violation")
	}

	fixed1 := linter.ApplyFixes(sql1, diags1)
	if !strings.HasPrefix(fixed1, "SELECT id FROM users;") {
		t.Errorf("expected fixed SQL to be 'SELECT id FROM users;', got %q", fixed1)
	}

	// 2. Test L002: SELECT *
	sql2 := "SELECT * FROM users;"
	diags2, _ := eng.Lint("test2.sql", sql2, d)
	hasL002 := false
	for _, diag := range diags2 {
		if diag.RuleID == "L002" {
			hasL002 = true
		}
	}
	if !hasL002 {
		t.Errorf("expected L002 (no-select-star) violation")
	}

	// 3. Test L006: UPDATE and DELETE without WHERE (Safety rule)
	sql3 := "UPDATE users SET active = false; DELETE FROM users;"
	diags3, _ := eng.Lint("test3.sql", sql3, d)
	l006Count := 0
	for _, diag := range diags3 {
		if diag.RuleID == "L006" {
			l006Count++
			if diag.Severity != linter.SeverityError {
				t.Errorf("expected L006 severity ERROR, got %s", diag.Severity)
			}
		}
	}
	if l006Count != 2 {
		t.Errorf("expected 2 L006 violations (for UPDATE and DELETE), got %d", l006Count)
	}

	// 4. Test L004 & L005: Operator and Comma Spacing
	sql4 := "SELECT id,name FROM users WHERE id=1;"
	diags4, _ := eng.Lint("test4.sql", sql4, d)
	hasL004 := false
	hasL005 := false
	for _, diag := range diags4 {
		if diag.RuleID == "L004" {
			hasL004 = true
		}
		if diag.RuleID == "L005" {
			hasL005 = true
		}
	}
	if !hasL004 {
		t.Errorf("expected L004 (operator-spacing) violation")
	}
	if !hasL005 {
		t.Errorf("expected L005 (comma-spacing) violation")
	}

	fixed4 := linter.ApplyFixes(sql4, diags4)
	if !strings.Contains(fixed4, "id, name") || !strings.Contains(fixed4, "id = 1") {
		t.Errorf("expected fixed spacing in %q", fixed4)
	}
}
