package c

import (
	"context"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCExecutables(t *testing.T) {
	cc, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc unavailable")
	}
	root, _ := filepath.Abs("../../..")
	for _, tc := range []struct{ name, want string }{{"loops", "20\n"}, {"collections", "42\n100\nlist index out of bounds\n"}, {"result", "Expected positive number\n"}, {"concurrency", "Todos os processos foram concluidos!"}} {
		t.Run(tc.name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join(root, "examples", tc.name+".liaf"))
			if err != nil {
				t.Fatal(err)
			}
			p := parser.New(lexer.New(string(b)), tc.name)
			m := p.ParseModule()
			if len(p.Diagnostics) > 0 {
				t.Fatal(p.Diagnostics)
			}
			src, err := Generate(m)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			file := filepath.Join(dir, "main.c")
			bin := filepath.Join(dir, "program.exe")
			os.WriteFile(file, []byte(src), 0600)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, cc, "-std=c99", "-pthread", file, "-o", bin)
			if b, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("compile: %v\n%s", err, b)
			}
			out, err := exec.CommandContext(ctx, bin).CombinedOutput()
			if err != nil {
				t.Fatalf("run: %v\n%s", err, out)
			}
			if !strings.Contains(strings.ReplaceAll(string(out), "\r\n", "\n"), tc.want) {
				t.Fatalf("output %q", out)
			}
		})
	}
}
