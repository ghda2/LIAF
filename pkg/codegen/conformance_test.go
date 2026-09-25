package codegen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"liaf/pkg/checker"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Suite de conformidade do nucleo basico (conformance/basics).
//
// Cada caso e um .liaf com cabecalho de comentarios:
//
//	;; issue: #020
//	;; status: pending | done
//	;; effects: io clock   (opcional; padrao "io")
//	;; stdin: texto        (opcional; uma linha por diretiva)
//	;; exit: 3             (opcional; padrao 0)
//	;; out: linha esperada (uma diretiva por linha de saida)
//
// Se o corpo nao comeca com "(module", ele e embrulhado num main.
// Casos "done" precisam passar. Casos "pending" sao pulados enquanto falham e
// derrubam o teste quando passam, para que o marcador seja trocado por "done".

type conformanceCase struct {
	path    string
	issue   string
	status  string
	effects string
	stdin   []string
	exit    int
	out     []string
	source  string
}

func parseConformanceCase(path string) (conformanceCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return conformanceCase{}, err
	}
	c := conformanceCase{path: path, effects: "io"}
	var body []string
	for line := range strings.SplitSeq(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, ";;") {
			body = append(body, line)
			continue
		}
		directive := strings.TrimSpace(strings.TrimPrefix(trimmed, ";;"))
		key, value, ok := strings.Cut(directive, ":")
		if !ok {
			continue
		}
		value = strings.TrimPrefix(value, " ")
		switch strings.TrimSpace(key) {
		case "issue":
			c.issue = value
		case "status":
			c.status = strings.TrimSpace(value)
		case "effects":
			c.effects = strings.TrimSpace(value)
		case "stdin":
			c.stdin = append(c.stdin, value)
		case "exit":
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return c, fmt.Errorf("exit invalido: %v", err)
			}
			c.exit = n
		case "out":
			c.out = append(c.out, value)
		}
	}
	if c.status != "pending" && c.status != "done" {
		return c, fmt.Errorf("status deve ser pending ou done, obtido %q", c.status)
	}
	src := strings.TrimSpace(strings.Join(body, "\n"))
	if !strings.HasPrefix(src, "(module") {
		src = fmt.Sprintf("(module conformance\n  (fn main (params) (returns void) (effects %s)\n    (body\n%s)))\n", c.effects, src)
	}
	c.source = src
	return c, nil
}

// runConformanceCase devolve nil quando o caso se comporta como esperado.
func runConformanceCase(t *testing.T, root string, c conformanceCase) error {
	p := parser.New(lexer.New(c.source), c.path)
	mod := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		d := p.Diagnostics[0]
		return fmt.Errorf("parse: [%s] %d:%d %s", d.Code, d.Line, d.Col, d.Message)
	}
	if _, diags := checker.Check(mod, c.path); len(diags) > 0 {
		return fmt.Errorf("check: %s", diags[0].String())
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte(New(mod).Generate()), 0600); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	bin := filepath.Join(dir, "prog.exe")
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, src)
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("go build: %v\n%s", err, out)
	}

	run := exec.CommandContext(ctx, bin)
	run.Dir = dir
	if len(c.stdin) > 0 {
		run.Stdin = strings.NewReader(strings.Join(c.stdin, "\n") + "\n")
	}
	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	exitCode := 0
	if err := run.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return fmt.Errorf("run: %v", err)
		}
		exitCode = exitErr.ExitCode()
	}

	got := strings.TrimRight(strings.ReplaceAll(stdout.String(), "\r\n", "\n"), "\n")
	want := strings.Join(c.out, "\n")
	if got != want || exitCode != c.exit {
		return fmt.Errorf("saida divergente\n--- esperado (exit %d)\n%s\n--- obtido (exit %d)\n%s\n--- stderr\n%s",
			c.exit, want, exitCode, got, firstLines(stderr.String(), 5))
	}
	return nil
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

func TestConformanceBasics(t *testing.T) {
	if testing.Short() {
		t.Skip("compila um binario por caso; pulado em -short")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(root, "conformance", "basics", "*.liaf"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("nenhum caso em conformance/basics")
	}
	for _, path := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(path), ".liaf"), func(t *testing.T) {
			t.Parallel()
			c, err := parseConformanceCase(path)
			if err != nil {
				t.Fatalf("cabecalho: %v", err)
			}
			err = runConformanceCase(t, root, c)
			switch {
			case c.status == "done" && err != nil:
				t.Fatalf("%s: %v", c.issue, err)
			case c.status == "pending" && err == nil:
				t.Fatalf("%s: caso passa; troque o marcador para status: done", c.issue)
			case c.status == "pending":
				t.Skipf("pendente (%s): %v", c.issue, firstLines(err.Error(), 1))
			}
		})
	}
}
