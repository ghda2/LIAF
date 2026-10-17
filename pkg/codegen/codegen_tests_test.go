package codegen

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

func TestCodegenNativeTestsLinear(t *testing.T) {
	liafSource := `
test "soma simples"
  let a = 10
  let b = 20
  assert(a + b == 30, "soma incorreta")
  assert-true(a < b)
  assert-false(a > b)
  assert-eq(a + b, 30, "30 esperado")
end

test "strings"
  assert-eq("liaf", "liaf", "strings iguais")
end
`
	lex := lexer.New(liafSource)
	p := parser.New(lex, "test_file.liaf")
	mod := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		t.Fatalf("erros no parse: %v", p.Diagnostics)
	}

	code := New(mod).Generate()
	if !strings.Contains(code, "rt.RunTests([]rt.TestCase{") {
		t.Fatalf("codegen nao gerou runner de testes: %s", code)
	}
	if !strings.Contains(code, `{Name: "soma simples", Fn: _liaf_test_0}`) {
		t.Fatalf("caso de teste 0 ausente: %s", code)
	}
	if !strings.Contains(code, `{Name: "strings", Fn: _liaf_test_1}`) {
		t.Fatalf("caso de teste 1 ausente: %s", code)
	}

	if testing.Short() {
		return
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	goSource := filepath.Join(dir, "main.go")
	if err := os.WriteFile(goSource, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	bin := filepath.Join(dir, "prog.exe")
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, goSource)
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	run := exec.CommandContext(ctx, bin)
	run.Dir = dir
	run.Env = append(os.Environ(), "LIAF_TEST_MODE=1")
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("execucao dos testes falhou: %v\n%s", err, out)
	}

	outStr := string(out)
	if !strings.Contains(outStr, "PASS: soma simples") || !strings.Contains(outStr, "PASS: strings") {
		t.Fatalf("saida inesperada dos testes: %s", outStr)
	}
}
