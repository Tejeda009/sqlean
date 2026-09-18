// Package formatter pkg/formatter/formatter.go
package formatter

import (
	"strings"

	"github.com/Tejeda009/sqlean/pkg/dialect"
	"github.com/Tejeda009/sqlean/pkg/lexer"
)

type Options struct {
	Indent              string // Indentation string (default "  ")
	UppercaseKeywords   bool   // True to convert keywords to UPPERCASE
	LinesBetweenQueries int    // Blank lines between multiple queries (default 1)
}

func DefaultOptions() Options {
	return Options{
		Indent:              "  ",
		UppercaseKeywords:   true,
		LinesBetweenQueries: 1,
	}
}

type Formatter struct {
	opts Options
}

func New(opts Options) *Formatter {
	if opts.Indent == "" {
		opts.Indent = "  "
	}
	if opts.LinesBetweenQueries <= 0 {
		opts.LinesBetweenQueries = 1
	}
	return &Formatter{opts: opts}
}

// Format takes a raw SQL string and formats it according to style rules.
func (f *Formatter) Format(sql string, d dialect.Dialect) (string, error) {
	if d == nil {
		d = dialect.GetDefault()
	}

	lex := lexer.New(sql, d)
	rawTokens := lex.TokenizeAll()

	// Filter raw whitespace tokens while preserving comments
	var tokens []lexer.Token
	for _, tok := range rawTokens {
		if tok.Type != lexer.TokenWhitespace {
			tokens = append(tokens, tok)
		}
	}

	if len(tokens) == 0 || (len(tokens) == 1 && tokens[0].Type == lexer.TokenEOF) {
		return "", nil
	}

	var sb strings.Builder
	indentLevel := 0
	parenDepth := 0
	subqueryDepths := make(map[int]bool) // tracks which opening parentheses start subqueries

	// Pre-pass: identify opening parentheses that start a subquery (SELECT)
	for i := 0; i < len(tokens)-1; i++ {
		if tokens[i].Literal == "(" {
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j].Type == lexer.TokenComment {
					continue
				}
				if strings.ToUpper(tokens[j].Literal) == "SELECT" {
					subqueryDepths[i] = true
				}
				break
			}
		}
	}

	var parenStack []int
	newlinePending := false
	clauseIndentPending := false
	inSelectProjection := false

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.Type == lexer.TokenEOF {
			break
		}

		upper := strings.ToUpper(tok.Literal)

		// 1. Comments
		if tok.Type == lexer.TokenComment {
			if strings.HasPrefix(tok.Literal, "--") || strings.HasPrefix(tok.Literal, "#") {
				if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "\n") {
					sb.WriteString(" ")
				}
				sb.WriteString(tok.Literal)
				sb.WriteString("\n")
				sb.WriteString(strings.Repeat(f.opts.Indent, indentLevel))
				continue
			}
			// Block comment /* ... */
			if sb.Len() > 0 && !strings.HasSuffix(sb.String(), " ") && !strings.HasSuffix(sb.String(), "\n") {
				sb.WriteString(" ")
			}
			sb.WriteString(tok.Literal)
			continue
		}

		// 2. Parentheses (Subqueries vs Expressions)
		if tok.Literal == "(" {
			parenStack = append(parenStack, i)
			parenDepth++
			isSub := subqueryDepths[i]

			if sb.Len() > 0 && !strings.HasSuffix(sb.String(), " ") && !strings.HasSuffix(sb.String(), "\n") {
				// Avoid space after function calls like COUNT(id)
				prev := tokens[i-1]
				prevUpper := strings.ToUpper(prev.Literal)
				if prev.Type == lexer.TokenIdentifier || prevUpper == "COUNT" || prevUpper == "SUM" ||
					prevUpper == "AVG" || prevUpper == "MIN" || prevUpper == "MAX" || prevUpper == "COALESCE" {
					// No space before (
				} else {
					sb.WriteString(" ")
				}
			}

			sb.WriteString("(")
			if isSub {
				indentLevel++
				sb.WriteString("\n")
			}
			continue
		}

		if tok.Literal == ")" {
			isSub := false
			if len(parenStack) > 0 {
				lastParenIdx := parenStack[len(parenStack)-1]
				parenStack = parenStack[:len(parenStack)-1]
				isSub = subqueryDepths[lastParenIdx]
			}
			parenDepth--

			if isSub {
				if indentLevel > 0 {
					indentLevel--
				}
				sb.WriteString("\n")
				sb.WriteString(strings.Repeat(f.opts.Indent, indentLevel))
			}
			sb.WriteString(")")
			continue
		}

		// 3. Major Keywords & Clauses
		if tok.Type == lexer.TokenKeyword {
			if f.opts.UppercaseKeywords {
				tok.Literal = upper
			}

			switch upper {
			case "SELECT":
				inSelectProjection = true
				if sb.Len() > 0 {
					if !strings.HasSuffix(sb.String(), "\n") {
						sb.WriteString("\n")
					}
				}
				sb.WriteString(strings.Repeat(f.opts.Indent, indentLevel))
				sb.WriteString(tok.Literal)
				clauseIndentPending = true
				continue

			case "FROM", "WHERE", "HAVING", "LIMIT", "OFFSET", "SET", "VALUES":
				inSelectProjection = false
				if sb.Len() > 0 {
					if !strings.HasSuffix(sb.String(), "\n") {
						sb.WriteString("\n")
					}
				}
				sb.WriteString(strings.Repeat(f.opts.Indent, indentLevel))
				sb.WriteString(tok.Literal)
				clauseIndentPending = true
				continue

			case "GROUP", "ORDER":
				inSelectProjection = false
				if i+1 < len(tokens) && strings.ToUpper(tokens[i+1].Literal) == "BY" {
					if sb.Len() > 0 {
						if !strings.HasSuffix(sb.String(), "\n") {
							sb.WriteString("\n")
						}
					}
					byTok := tokens[i+1]
					byLit := byTok.Literal
					if f.opts.UppercaseKeywords {
						byLit = "BY"
					}
					sb.WriteString(strings.Repeat(f.opts.Indent, indentLevel))
					sb.WriteString(tok.Literal)
					sb.WriteString(" ")
					sb.WriteString(byLit)
					i++ // consume BY
					clauseIndentPending = true
					continue
				}

			case "INNER", "LEFT", "RIGHT", "FULL", "CROSS":
				inSelectProjection = false
				if i+1 < len(tokens) && strings.ToUpper(tokens[i+1].Literal) == "JOIN" {
					if sb.Len() > 0 {
						if !strings.HasSuffix(sb.String(), "\n") {
							sb.WriteString("\n")
						}
					}
					joinLit := tokens[i+1].Literal
					if f.opts.UppercaseKeywords {
						joinLit = "JOIN"
					}
					sb.WriteString(strings.Repeat(f.opts.Indent, indentLevel))
					sb.WriteString(tok.Literal)
					sb.WriteString(" ")
					sb.WriteString(joinLit)
					i++ // consume JOIN
					clauseIndentPending = true
					continue
				}

			case "JOIN":
				inSelectProjection = false
				if sb.Len() > 0 {
					if !strings.HasSuffix(sb.String(), "\n") {
						sb.WriteString("\n")
					}
				}
				sb.WriteString(strings.Repeat(f.opts.Indent, indentLevel))
				sb.WriteString(tok.Literal)
				clauseIndentPending = true
				continue

			case "INSERT":
				inSelectProjection = false
				if sb.Len() > 0 {
					if !strings.HasSuffix(sb.String(), "\n") {
						sb.WriteString("\n")
					}
				}
				sb.WriteString(strings.Repeat(f.opts.Indent, indentLevel))
				sb.WriteString(tok.Literal)
				if i+1 < len(tokens) && strings.ToUpper(tokens[i+1].Literal) == "INTO" {
					intoLit := tokens[i+1].Literal
					if f.opts.UppercaseKeywords {
						intoLit = "INTO"
					}
					sb.WriteString(" ")
					sb.WriteString(intoLit)
					i++
				}
				continue

			case "UPDATE", "DELETE":
				inSelectProjection = false
				if sb.Len() > 0 {
					if !strings.HasSuffix(sb.String(), "\n") {
						sb.WriteString("\n")
					}
				}
				sb.WriteString(strings.Repeat(f.opts.Indent, indentLevel))
				sb.WriteString(tok.Literal)
				if upper == "DELETE" && i+1 < len(tokens) && strings.ToUpper(tokens[i+1].Literal) == "FROM" {
					fromLit := tokens[i+1].Literal
					if f.opts.UppercaseKeywords {
						fromLit = "FROM"
					}
					sb.WriteString(" ")
					sb.WriteString(fromLit)
					i++
				}
				continue

			case "AND", "OR":
				if parenDepth == 0 || (len(parenStack) > 0 && subqueryDepths[parenStack[len(parenStack)-1]]) {
					sb.WriteString("\n")
					sb.WriteString(strings.Repeat(f.opts.Indent, indentLevel+1))
					sb.WriteString(tok.Literal)
					sb.WriteString(" ")
					continue
				}
			}
		}

		// 4. Pending clause indentation / newlines
		if clauseIndentPending {
			sb.WriteString("\n")
			sb.WriteString(strings.Repeat(f.opts.Indent, indentLevel+1))
			clauseIndentPending = false
		} else if newlinePending {
			sb.WriteString("\n")
			sb.WriteString(strings.Repeat(f.opts.Indent, indentLevel+1))
			newlinePending = false
		} else if i > 0 && sb.Len() > 0 && !strings.HasSuffix(sb.String(), " ") && !strings.HasSuffix(sb.String(), "\n") && !strings.HasSuffix(sb.String(), "(") {
			if tok.Literal != "," && tok.Literal != ";" && tok.Literal != "." && tok.Literal != "::" {
				prevTok := tokens[i-1]
				if prevTok.Literal != "." && prevTok.Literal != "::" {
					sb.WriteString(" ")
				}
			}
		}

		// 5. Output literal
		sb.WriteString(tok.Literal)

		// 6. Punctuation
		if tok.Literal == "," {
			if inSelectProjection && parenDepth == 0 {
				newlinePending = true
			}
		}

		if tok.Literal == ";" {
			inSelectProjection = false
			sb.WriteString("\n")
			if f.opts.LinesBetweenQueries > 1 {
				sb.WriteString(strings.Repeat("\n", f.opts.LinesBetweenQueries-1))
			}
		}
	}

	res := strings.TrimSpace(sb.String()) + "\n"
	return res, nil
}
