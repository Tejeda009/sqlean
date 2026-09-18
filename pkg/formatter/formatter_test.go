// pkg/formatter/formatter_test.go
package formatter_test

import (
	"strings"
	"testing"

	"github.com/Tejeda009/sqlean/pkg/dialect"
	"github.com/Tejeda009/sqlean/pkg/formatter"
)

func TestFormatterSelect(t *testing.T) {
	raw := "select id, name, email from users where id = 1 and status = 'active' order by name desc;"
	fmtEngine := formatter.New(formatter.DefaultOptions())
	out, err := fmtEngine.Format(raw, dialect.GetDefault())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedParts := []string{
		"SELECT",
		"  id,",
		"  name,",
		"  email",
		"FROM",
		"  users",
		"WHERE",
		"  id = 1",
		"  AND status = 'active'",
		"ORDER BY",
		"  name DESC;",
	}

	for _, part := range expectedParts {
		if !strings.Contains(out, part) {
			t.Errorf("expected formatted output to contain %q, but got:\n%s", part, out)
		}
	}
}

func TestFormatterJoinAndSubquery(t *testing.T) {
	raw := "SELECT u.id, o.total FROM users AS u LEFT JOIN orders AS o ON u.id = o.user_id WHERE u.id IN (SELECT user_id FROM vip_users);"
	fmtEngine := formatter.New(formatter.DefaultOptions())
	out, err := fmtEngine.Format(raw, dialect.GetDefault())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "LEFT JOIN") {
		t.Errorf("expected LEFT JOIN in output, got:\n%s", out)
	}
	if !strings.Contains(out, "vip_users") || !strings.Contains(out, "user_id") {
		t.Errorf("expected nested subquery, got:\n%s", out)
	}
}

func TestFormatterInsertUpdateDelete(t *testing.T) {
	fmtEngine := formatter.New(formatter.DefaultOptions())

	// INSERT
	insSQL := "insert into products (name, price) values ('Keyboard', 99.99);"
	insOut, err := fmtEngine.Format(insSQL, dialect.GetDefault())
	if err != nil {
		t.Fatalf("error formatting insert: %v", err)
	}
	if !strings.Contains(insOut, "INSERT INTO products") || !strings.Contains(insOut, "VALUES") {
		t.Errorf("unexpected insert format:\n%s", insOut)
	}

	// UPDATE
	updSQL := "update products set price = 89.99 where id = 5;"
	updOut, err := fmtEngine.Format(updSQL, dialect.GetDefault())
	if err != nil {
		t.Fatalf("error formatting update: %v", err)
	}
	if !strings.Contains(updOut, "UPDATE products") || !strings.Contains(updOut, "SET") || !strings.Contains(updOut, "WHERE") {
		t.Errorf("unexpected update format:\n%s", updOut)
	}

	// DELETE
	delSQL := "delete from products where price <= 0;"
	delOut, err := fmtEngine.Format(delSQL, dialect.GetDefault())
	if err != nil {
		t.Fatalf("error formatting delete: %v", err)
	}
	if !strings.Contains(delOut, "DELETE FROM products") || !strings.Contains(delOut, "WHERE") {
		t.Errorf("unexpected delete format:\n%s", delOut)
	}
}

func TestFormatterCommentsPreservation(t *testing.T) {
	sql := "-- filters only active users\nSELECT id FROM users;"
	fmtEngine := formatter.New(formatter.DefaultOptions())
	out, err := fmtEngine.Format(sql, dialect.GetDefault())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "-- filters only active users") {
		t.Errorf("comment was dropped:\n%s", out)
	}
}
