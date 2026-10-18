package codegen

import (
	"context"
	"liaf/pkg/loader"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCodegenStdContentIntegration(t *testing.T) {
	liafSource := `
(module test_content
  (import "std/content")

  (test "slugify e parse"
    (let slug str (content-slugify "Meu Artigo IA First"))
    (assert-eq slug "meu-artigo-ia-first" "slug deve ser correto")

    (let raw str "---\ntitle: \"Titulo\"\n---\nCorpo")
    (match (content-parse raw)
      (ok j
        (assert-true (gt (str-len j) 0)))
      (err e
        (assert false e))))

  (fn main (params) (returns void) (effects io)
    (body
      (println (concat "slug: " (content-slugify "Artigo 100% IA"))))))
`
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "main.liaf")
	if err := os.WriteFile(sourceFile, []byte(liafSource), 0600); err != nil {
		t.Fatal(err)
	}

	mod, diags := loader.Load(sourceFile)
	if len(diags) > 0 {
		t.Fatalf("erros no loader: %v", diags)
	}

	code := New(mod).Generate()
	if !strings.Contains(code, "rt.ContentSlugify") {
		t.Fatalf("codegen nao gerou chamada para rt.ContentSlugify: %s", code)
	}

	if testing.Short() {
		return
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	goSource := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goSource, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	bin := filepath.Join(tmpDir, "prog.exe")
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, goSource)
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build falhou: %v\n%s", err, out)
	}

	// Executa em modo normal
	run := exec.CommandContext(ctx, bin)
	run.Dir = tmpDir
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("execucao normal falhou: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "slug: artigo-100-ia") {
		t.Fatalf("saida inesperada: %s", string(out))
	}

	// Executa em modo teste nativo (LIAF_TEST_MODE=1)
	runTest := exec.CommandContext(ctx, bin)
	runTest.Dir = tmpDir
	runTest.Env = append(os.Environ(), "LIAF_TEST_MODE=1")
	outTest, err := runTest.CombinedOutput()
	if err != nil {
		t.Fatalf("execucao de teste falhou: %v\n%s", err, outTest)
	}
	if !strings.Contains(string(outTest), "PASS: slugify e parse") {
		t.Fatalf("saida de teste inesperada: %s", string(outTest))
	}
}
