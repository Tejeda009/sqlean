// Package dialect pkg/dialect/dialect.go
package dialect

import (
	"fmt"
	"strings"
	"sync"
)

// CustomTokenType categorizes dialect-specific lexical tokens.
type CustomTokenType string

const (
	CustomTokenPlaceholder CustomTokenType = "PLACEHOLDER"
	CustomTokenOperator    CustomTokenType = "OPERATOR"
	CustomTokenIdentifier  CustomTokenType = "IDENTIFIER"
	CustomTokenLiteral     CustomTokenType = "LITERAL"
	CustomTokenComment     CustomTokenType = "COMMENT"
)

// CustomToken represents a token specific to a database dialect.
type CustomToken struct {
	Type    CustomTokenType
	Literal string
}

// Dialect abstracts engine-specific SQL syntax differences.
type Dialect interface {
	Name() string
	IsKeyword(ident string) bool
	QuoteIdentifier(ident string) string
	// ScanCustomToken scans custom constructs (e.g. Postgres $1, ::, MySQL @var, #)
	ScanCustomToken(remaining string) (CustomToken, bool)
}

var (
	mu       sync.RWMutex
	registry = make(map[string]Dialect)
)

// Register adds a dialect to the global registry thread-safely.
func Register(d Dialect) {
	mu.Lock()
	defer mu.Unlock()
	registry[strings.ToLower(d.Name())] = d
}

// Get fetches a dialect by name.
func Get(name string) (Dialect, error) {
	mu.RLock()
	defer mu.RUnlock()
	d, ok := registry[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("unknown dialect: %q (supported: ansi, postgres, mysql)", name)
	}
	return d, nil
}

// GetDefault returns the fallback ANSI dialect.
func GetDefault() Dialect {
	d, err := Get("ansi")
	if err != nil {
		return NewANSIDialect()
	}
	return d
}

// AvailableDialects returns a list of all registered dialect names.
func AvailableDialects() []string { // this may be unused
	mu.RLock()
	defer mu.RUnlock()
	list := make([]string, 0, len(registry))
	for name := range registry {
		list = append(list, name)
	}
	return list
}
