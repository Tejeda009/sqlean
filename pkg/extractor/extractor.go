// Package extractor pkg/extractor/extractor.go
package extractor

import (
	goast "go/ast"
	goparser "go/parser"
	gotoken "go/token"
	"os"
	"sort"
	"strings"
)

// ExtractedSQL contains the SQL extracted from the exact mapping of his position in .go file
type ExtractedSQL struct {
	Query       string `json:"query"`
	StartOffset int    `json:"start_offset"`  // byte offset of the first SQL character
	EndOffset   int    `json:"end_offset"`    //byte offset of the end of the SQL
	StartLine   int    `json:"start_line"`    // 1-based line in the go file
	StartCol    int    `json:"start_col"`     // 1-based column ''
	IsRawString bool   `json:"is_raw_string"` // true if defined by bactick
}

// Replacement defines the substitution of a sql fragment inside the go file
type Replacement struct {
	StartOffset int
	EndOffset   int
	NewSQL      string
	IsRawString bool
}

// ExtractFromGoSource analizes the source and finds sql query incorporated  .
func ExtractFromGoSource(filePath string, src []byte) ([]ExtractedSQL, error) {
	fset := gotoken.NewFileSet()
	node, err := goparser.ParseFile(fset, filePath, src, goparser.ParseComments)
	if err != nil {
		return nil, err
	}

	var extracted []ExtractedSQL

	//map of the comments to find things like // sql or similar
	commentLines := make(map[int]bool)
	for _, cg := range node.Comments {
		for _, c := range cg.List {
			text := strings.ToLower(c.Text)
			if strings.Contains(text, "sql") {
				pos := fset.Position(c.Pos())
				commentLines[pos.Line] = true
				commentLines[pos.Line+1] = true
			}
		}
	}

	goast.Inspect(node, func(n goast.Node) bool {
		//Finds literal strings
		lit, ok := n.(*goast.BasicLit)
		if !ok || lit.Kind != gotoken.STRING {
			return true
		}

		isRaw := strings.HasPrefix(lit.Value, "`") && strings.HasSuffix(lit.Value, "`")
		cleanVal := lit.Value
		if isRaw && len(lit.Value) >= 2 {
			cleanVal = lit.Value[1 : len(lit.Value)-1]
		} else if strings.HasPrefix(lit.Value, "\"") && strings.HasSuffix(lit.Value, "\"") && len(lit.Value) >= 2 {
			cleanVal = lit.Value[1 : len(lit.Value)-1]
		}

		trimmed := strings.TrimSpace(strings.ToUpper(cleanVal))
		pos := fset.Position(lit.Pos())

		hasSQLDirective := commentLines[pos.Line]

		isSQL := hasSQLDirective ||
			strings.HasPrefix(trimmed, "SELECT ") ||
			strings.HasPrefix(trimmed, "INSERT INTO ") ||
			strings.HasPrefix(trimmed, "UPDATE ") ||
			strings.HasPrefix(trimmed, "DELETE FROM ") ||
			strings.HasPrefix(trimmed, "CREATE TABLE ") ||
			strings.HasPrefix(trimmed, "ALTER TABLE ") ||
			strings.HasPrefix(trimmed, "DROP TABLE ") ||
			strings.HasPrefix(trimmed, "WITH ")

		if isSQL && len(cleanVal) > 0 {
			// lit.Pos() points to the first character
			startOff := int(lit.Pos()) - 1 + 1 // 0-based index of the inside
			endOff := int(lit.End()) - 1 - 1   // 0-based index end of the inside

			extracted = append(extracted, ExtractedSQL{
				Query:       cleanVal,
				StartOffset: startOff,
				EndOffset:   endOff,
				StartLine:   pos.Line,
				StartCol:    pos.Column + 1,
				IsRawString: isRaw,
			})
		}

		return true
	})

	return extracted, nil
}

// RewriteGoSource apllies changes to the sql in the source code without breaking the syntax (hopefully)
func RewriteGoSource(originalSrc []byte, replacements []Replacement) []byte {
	if len(replacements) == 0 {
		return originalSrc
	}

	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].StartOffset > replacements[j].StartOffset
	})

	result := string(originalSrc)

	for _, rep := range replacements {
		if rep.StartOffset >= 0 && rep.EndOffset <= len(result) && rep.StartOffset <= rep.EndOffset {
			newContent := rep.NewSQL

			if !rep.IsRawString && strings.Contains(newContent, "\n") && !strings.Contains(newContent, "`") {

				startWithQuote := rep.StartOffset - 1
				endWithQuote := rep.EndOffset + 1
				if startWithQuote >= 0 && endWithQuote <= len(result) &&
					result[startWithQuote] == '"' && result[endWithQuote-1] == '"' {
					result = result[:startWithQuote] + "`" + strings.TrimSpace(newContent) + "`" + result[endWithQuote:]
					continue
				}
			}

			if rep.IsRawString {
				newContent = "\n" + strings.TrimSpace(newContent) + "\n"
			} else {
				newContent = strings.TrimSpace(newContent)
			}
			result = result[:rep.StartOffset] + newContent + result[rep.EndOffset:]
		}
	}

	return []byte(result)
}

// RewriteGoFile aplies changes and writes them to the disk
func RewriteGoFile(filePath string, originalSrc []byte, replacements []Replacement) error {
	modified := RewriteGoSource(originalSrc, replacements)
	return os.WriteFile(filePath, modified, 0644)
}
