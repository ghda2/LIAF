package codegen

import (
	"context"
	"liaf/pkg/ast"
	"liaf/pkg/checker"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecutableExamples(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ file, output string }{
		{"loops", "20\n"}, {"collections", "42\n100\nlist index out of bounds\n"},
		{"fs_json", "alice\nremovido\n"}, {"result", "Expected positive number\n"}, {"math", ""},
	} {
		t.Run(tc.file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, "examples", tc.file+".liaf"))
			if err != nil {
				t.Fatal(err)
			}
			p := parser.New(lexer.New(string(data)), tc.file)
			mod := p.ParseModule()
			if len(p.Diagnostics) > 0 {
				t.Fatal(p.Diagnostics)
			}
			_, diags := checker.Check(mod, tc.file)
			if len(diags) > 0 {
				t.Fatal(diags)
			}
			formatted := ast.Format(mod)
			p = parser.New(lexer.New(formatted), tc.file)
			round := p.ParseModule()
			if len(p.Diagnostics) > 0 {
				t.Fatalf("format roundtrip: %s\n%+v", formatted, p.Diagnostics)
			}
			if ast.Format(round) != formatted {
				t.Fatal("formatter is not idempotent")
			}
			dir := t.TempDir()
			source := filepath.Join(dir, "main.go")
			if err = os.WriteFile(source, []byte(New(mod).Generate()), 0600); err != nil {
				t.Fatal(err)
			}
			bin := filepath.Join(dir, "example.exe")
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, source)
			cmd.Dir = root
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("build: %v\n%s", err, output)
			}
			cmd = exec.CommandContext(ctx, bin)
			cmd.Dir = dir
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("run: %v\n%s", err, output)
			}
			if tc.output != "" && strings.ReplaceAll(string(output), "\r\n", "\n") != tc.output {
				t.Fatalf("got %q, want %q", output, tc.output)
			}
		})
	}
}
