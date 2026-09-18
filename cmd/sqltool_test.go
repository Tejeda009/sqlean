// cmd/sqltool_test.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/Tejeda009/sqlean/pkg/lexer"
	"github.com/Tejeda009/sqlean/pkg/linter"
)

func TestOutputDiagnostics(t *testing.T) {
	diags := []linter.Diagnostic{
		{
			RuleID:   "L001",
			Message:  "Keyword in lowercase",
			Severity: linter.SeverityWarning,
			File:     "test.sql",
			Start:    lexer.Position{Line: 1, Column: 1, Offset: 0},
			End:      lexer.Position{Line: 1, Column: 7, Offset: 6},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputDiagnostics(diags, "json")

	err := w.Close()
	if err != nil {
		fmt.Println("error:", err.Error())
		return
	}
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	var decoded []linter.Diagnostic
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode JSON output: %v", err)
	}

	if len(decoded) != 1 || decoded[0].RuleID != "L001" {
		t.Errorf("unexpected decoded JSON: %+v", decoded)
	}
}
