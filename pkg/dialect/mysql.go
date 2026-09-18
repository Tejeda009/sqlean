// Package dialect pkg/dialect/mysql.go
package dialect

import (
	"strings"
	"unicode"
)

type MySQLDialect struct {
	*ANSIDialect
}

func init() {
	Register(NewMySQLDialect())
}

func NewMySQLDialect() *MySQLDialect {
	base := NewANSIDialect()
	mysqlKeywords := []string{
		"AUTO_INCREMENT", "FORCE", "USE", "IGNORE", "STRAIGHT_JOIN",
		"SHOW", "DATABASES", "TABLES", "DESCRIBE", "EXPLAIN",
		"REPLACE", "DUPLICATE", "KEY", "UNSIGNED", "ZEROFILL",
		"ENUM", "SET", "ENGINE", "CHARSET", "COLLATE", "NOW",
		"IFNULL", "IF", "REGEXP", "RLIKE", "XOR", "BINARY",
		"VARBINARY", "TINYINT", "SMALLINT", "MEDIUMINT", "BIGINT",
	}
	for _, kw := range mysqlKeywords {
		base.keywords[kw] = struct{}{}
	}
	return &MySQLDialect{ANSIDialect: base}
}

func (d *MySQLDialect) Name() string {
	return "mysql"
}

func (d *MySQLDialect) QuoteIdentifier(ident string) string {
	return "`" + ident + "`"
}

// ScanCustomToken scans MySQL specific constructs:
// 1. Hash line comments: #
// 2. User and system variables: @var, @@sysvar
// 3. Positional placeholder: ?
func (d *MySQLDialect) ScanCustomToken(remaining string) (CustomToken, bool) {
	if len(remaining) == 0 {
		return CustomToken{}, false
	}

	// 1. Hash comment #
	if remaining[0] == '#' {
		before, _, ok := strings.Cut(remaining, "\n")
		if !ok {
			return CustomToken{Type: CustomTokenComment, Literal: remaining}, true
		}
		return CustomToken{Type: CustomTokenComment, Literal: before}, true
	}

	// 2. Session and global variables @var or @@global_var
	if remaining[0] == '@' {
		i := 1
		if i < len(remaining) && remaining[i] == '@' {
			i++
		}
		for i < len(remaining) && (unicode.IsLetter(rune(remaining[i])) || unicode.IsDigit(rune(remaining[i])) || remaining[i] == '_' || remaining[i] == '.') {
			i++
		}
		return CustomToken{Type: CustomTokenIdentifier, Literal: remaining[:i]}, true
	}

	// 3. Positional placeholder '?'
	if remaining[0] == '?' {
		return CustomToken{Type: CustomTokenPlaceholder, Literal: "?"}, true
	}

	return CustomToken{}, false
}
