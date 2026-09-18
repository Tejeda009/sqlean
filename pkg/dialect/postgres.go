// Package dialect pkg/dialect/postgres.go
package dialect

import (
	"strings"
	"unicode"
)

type PostgresDialect struct {
	*ANSIDialect
}

func init() {
	Register(NewPostgresDialect())
}

func NewPostgresDialect() *PostgresDialect {
	base := NewANSIDialect()
	pgKeywords := []string{
		"RETURNING", "ILIKE", "CONFLICT", "DO", "NOTHING",
		"LATERAL", "WINDOW", "PARTITION", "OVER", "VACUUM", "ANALYZE",
		"CASCADE", "RESTRICT", "FETCH", "FIRST", "ROWS", "ONLY",
		"NOWAIT", "SKIP", "LOCKED", "UNLOGGED", "MATERIALIZED",
		"TABLESPACE", "SCHEMA", "EXTENSION", "GENERATE_SERIES",
		"SIMILAR", "TO", "ARRAY", "JSONB", "TIMESTAMPTZ",
	}
	for _, kw := range pgKeywords {
		base.keywords[kw] = struct{}{}
	}
	return &PostgresDialect{ANSIDialect: base}
}

func (d *PostgresDialect) Name() string {
	return "postgres"
}

// ScanCustomToken scans Postgres specific constructs:
// 1. Positional parameters: $1, $2, $12
// 2. Type cast operator: ::text
// 3. Dollar-quoted strings: $$text$$ or $tag$text$tag$
// 4. Regex operators: ~, ~*, !~, !~*
func (d *PostgresDialect) ScanCustomToken(remaining string) (CustomToken, bool) {
	if len(remaining) == 0 {
		return CustomToken{}, false
	}

	// 1. Cast operator ::
	if strings.HasPrefix(remaining, "::") {
		return CustomToken{
			Type:    CustomTokenOperator,
			Literal: "::",
		}, true
	}

	// 2. Positional placeholders $1, $2...
	if remaining[0] == '$' && len(remaining) > 1 && unicode.IsDigit(rune(remaining[1])) {
		i := 1
		for i < len(remaining) && unicode.IsDigit(rune(remaining[i])) {
			i++
		}
		return CustomToken{
			Type:    CustomTokenPlaceholder,
			Literal: remaining[:i],
		}, true
	}

	// 3. Dollar-quoted strings ($$...$$ or $tag$...$tag$)
	if remaining[0] == '$' {
		tagEnd := strings.Index(remaining[1:], "$")
		if tagEnd != -1 {
			tag := remaining[:tagEnd+2]
			validTag := true
			for _, r := range tag[1 : len(tag)-1] {
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
					validTag = false
					break
				}
			}
			if validTag {
				bodyStart := len(tag)
				closingIdx := strings.Index(remaining[bodyStart:], tag)
				if closingIdx != -1 {
					totalLen := bodyStart + closingIdx + len(tag)
					return CustomToken{
						Type:    CustomTokenLiteral,
						Literal: remaining[:totalLen],
					}, true
				}
			}
		}
	}

	// 4. Postgres Regex operators
	if strings.HasPrefix(remaining, "!~*") {
		return CustomToken{Type: CustomTokenOperator, Literal: "!~*"}, true
	}
	if strings.HasPrefix(remaining, "!~") {
		return CustomToken{Type: CustomTokenOperator, Literal: "!~"}, true
	}
	if strings.HasPrefix(remaining, "~*") {
		return CustomToken{Type: CustomTokenOperator, Literal: "~*"}, true
	}
	if strings.HasPrefix(remaining, "~") {
		return CustomToken{Type: CustomTokenOperator, Literal: "~"}, true
	}

	return CustomToken{}, false
}
