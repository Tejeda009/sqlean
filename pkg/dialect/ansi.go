// Package dialect pkg/dialect/ansi.go
package dialect

import "strings"

type ANSIDialect struct {
	keywords map[string]struct{}
}

func init() {
	Register(NewANSIDialect())
}

func NewANSIDialect() *ANSIDialect {
	kws := []string{
		"SELECT", "FROM", "WHERE", "GROUP", "BY", "HAVING", "ORDER",
		"JOIN", "INNER", "LEFT", "RIGHT", "FULL", "OUTER", "CROSS", "ON", "USING",
		"INSERT", "INTO", "VALUES", "UPDATE", "SET", "DELETE",
		"CREATE", "TABLE", "ALTER", "DROP", "TRUNCATE", "INDEX", "VIEW",
		"AND", "OR", "NOT", "IN", "IS", "NULL", "AS", "DISTINCT", "ALL",
		"UNION", "INTERSECT", "EXCEPT", "LIMIT", "OFFSET",
		"CASE", "WHEN", "THEN", "ELSE", "END",
		"TRUE", "FALSE", "DEFAULT", "PRIMARY", "KEY", "FOREIGN", "REFERENCES",
		"CONSTRAINT", "UNIQUE", "CHECK", "ASC", "DESC", "BETWEEN", "LIKE", "EXISTS",
		"WITH", "RECURSIVE", "CAST", "COALESCE", "NULLIF", "COUNT", "SUM", "AVG", "MIN", "MAX",
	}
	m := make(map[string]struct{}, len(kws))
	for _, kw := range kws {
		m[kw] = struct{}{}
	}
	return &ANSIDialect{keywords: m}
}

func (d *ANSIDialect) Name() string {
	return "ansi"
}

func (d *ANSIDialect) IsKeyword(ident string) bool {
	_, ok := d.keywords[strings.ToUpper(ident)]
	return ok
}

func (d *ANSIDialect) QuoteIdentifier(ident string) string {
	return `"` + ident + `"`
}

func (d *ANSIDialect) ScanCustomToken(_ string) (CustomToken, bool) {
	return CustomToken{}, false
}
