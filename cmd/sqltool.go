// cmd/sqltool.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Tejeda009/sqlean/pkg/dialect"
	"github.com/Tejeda009/sqlean/pkg/extractor"
	"github.com/Tejeda009/sqlean/pkg/formatter"
	"github.com/Tejeda009/sqlean/pkg/linter"
	"github.com/Tejeda009/sqlean/pkg/linter/rules"
)

func main() {
	writeFlag := flag.Bool("w", false, "Write result to (source) file instead of stdout")
	checkFlag := flag.Bool("check", false, "Run in CI dry-run mode: exit 1 if unformatted or violations found")
	dialectFlag := flag.String("dialect", "ansi", "SQL dialect (ansi, postgres, mysql)")
	formatFlag := flag.String("format", "text", "Diagnostic output format: 'text' or 'json'")
	fixFlag := flag.Bool("fix", false, "Automatically apply linter fixes where possible")
	flag.Parse()

	d, err := dialect.Get(*dialectFlag)
	if err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if err != nil {
			fmt.Println("i love go: ", err.Error())
		}
		os.Exit(2)
	}

	fmtEngine := formatter.New(formatter.Options{
		Indent:              "  ",
		UppercaseKeywords:   true,
		LinesBetweenQueries: 1,
	})

	lintEngine := linter.NewEngine()
	lintEngine.RegisterRules(rules.AllRules()...)

	args := flag.Args()

	// Handle STDIN / STDOUT mode (editor formatting integration)
	if len(args) == 0 || args[0] == "-" {
		inputBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			_, err := fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			if err != nil {
				fmt.Println("go errors are incredible: ", err.Error())
			}
			os.Exit(2)
		}

		rawSQL := string(inputBytes)
		diags, err := lintEngine.Lint("<stdin>", rawSQL, d)
		if err != nil {
			_, err := fmt.Fprintf(os.Stderr, "Lint error: %v\n", err)
			if err != nil {
				fmt.Println("AUGE (Another Useless Go Error): ", err.Error())
			}
			os.Exit(2)
		}

		processedSQL := rawSQL
		if *fixFlag {
			processedSQL = linter.ApplyFixes(processedSQL, diags)
		}

		formattedSQL, err := fmtEngine.Format(processedSQL, d)
		if err != nil {
			_, err := fmt.Fprintf(os.Stderr, "Format error: %v\n", err)
			if err != nil {
				fmt.Println("I Love AUGE (read line 64): ", err.Error())
			}
			os.Exit(2)
		}

		if *checkFlag {
			hasViolations := len(diags) > 0 || formattedSQL != rawSQL
			outputDiagnostics(diags, *formatFlag)
			if hasViolations {
				os.Exit(1)
			}
			os.Exit(0)
		}

		fmt.Print(formattedSQL)
		os.Exit(0)
	}

	// File / Directory processing mode
	hasViolations := false
	var allDiagnostics []linter.Diagnostic

	for _, target := range args {
		info, err := os.Stat(target)
		if err != nil {
			_, err := fmt.Fprintf(os.Stderr, "Path not found: %s (%v)\n", target, err)
			if err != nil {
				fmt.Println("Go errors are my life: ", err.Error())
			}
			os.Exit(2)
		}

		walkFunc := func(path string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() {
				return err
			}

			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".sql" {
				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				origStr := string(content)
				diags, err := lintEngine.Lint(path, origStr, d)
				if err == nil && len(diags) > 0 {
					allDiagnostics = append(allDiagnostics, diags...)
					hasViolations = true
				}

				workStr := origStr
				if *fixFlag {
					workStr = linter.ApplyFixes(workStr, diags)
				}

				formatted, err := fmtEngine.Format(workStr, d)
				if err != nil {
					return err
				}

				if formatted != origStr {
					hasViolations = true
					if *writeFlag {
						if err := os.WriteFile(path, []byte(formatted), fi.Mode()); err != nil {
							return err
						}
					}
				}
			} else if ext == ".go" {
				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				sqls, err := extractor.ExtractFromGoSource(path, content)
				if err != nil {
					return err
				}

				var replacements []extractor.Replacement

				for _, item := range sqls {
					diags, err := lintEngine.Lint(path, item.Query, d)
					if err == nil && len(diags) > 0 {
						// Adjust line numbers relative to outer Go file
						for i := range diags {
							diags[i].Start.Line += item.StartLine - 1
							diags[i].End.Line += item.StartLine - 1
						}
						allDiagnostics = append(allDiagnostics, diags...)
						hasViolations = true
					}

					curQuery := item.Query
					if *fixFlag {
						curQuery = linter.ApplyFixes(curQuery, diags)
					}

					formatted, err := fmtEngine.Format(curQuery, d)
					if err == nil {
						trimmedFmt := strings.TrimSpace(formatted)
						trimmedOrig := strings.TrimSpace(item.Query)
						if trimmedFmt != trimmedOrig {
							hasViolations = true
							replacements = append(replacements, extractor.Replacement{
								StartOffset: item.StartOffset,
								EndOffset:   item.EndOffset,
								NewSQL:      formatted,
								IsRawString: item.IsRawString,
							})
						}
					}
				}

				if *writeFlag && len(replacements) > 0 {
					if err := extractor.RewriteGoFile(path, content, replacements); err != nil {
						return err
					}
				}
			}

			return nil
		}

		if info.IsDir() {
			if err := filepath.Walk(target, walkFunc); err != nil {
				_, err := fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
				if err != nil {
					fmt.Println("maybe this is a YAUGE (Yet Another Useless Go Error): ", err.Error())
				}
				os.Exit(2)
			}
		} else {
			if err := walkFunc(target, info, nil); err != nil {
				_, err := fmt.Fprintf(os.Stderr, "Error processing file %s: %v\n", target, err)
				if err != nil {
					fmt.Println("Woah, another go error: ", err.Error())
				}
				os.Exit(2)
			}
		}
	}

	outputDiagnostics(allDiagnostics, *formatFlag)

	if *checkFlag && hasViolations {
		os.Exit(1)
	}

	os.Exit(0)
}

func outputDiagnostics(diags []linter.Diagnostic, format string) {
	if len(diags) == 0 {
		return
	}

	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(diags)
		return
	}

	for _, d := range diags {
		fmt.Printf("%s:%d:%d: [%s] (%s) %s\n",
			d.File, d.Start.Line, d.Start.Column, d.RuleID, d.Severity, d.Message)
	}
}
