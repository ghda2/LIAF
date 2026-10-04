package codegen_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"liaf/pkg/codegen"
	"liaf/pkg/loader"
)

func TestCodegenVisibilityPubAndPrivate(t *testing.T) {
	dir := t.TempDir()

	mathFile := filepath.Join(dir, "math_utils.liaf")
	mathContent := `(module math_utils
  (fn helper (params (n int)) (returns int) (effects)
    (body (return (add n 10))))

  (pub fn calcular (params (n int)) (returns int) (effects)
    (body (return (helper n)))))`
	if err := os.WriteFile(mathFile, []byte(mathContent), 0600); err != nil {
		t.Fatal(err)
	}

	mainFile := filepath.Join(dir, "main.liaf")
	mainContent := `(module main
  (import "./math_utils.liaf")

  (fn helper (params (n int)) (returns int) (effects)
    (body (return (add n 100))))

  (fn main (params) (returns void) (effects io)
    (body
      (println (concat "calculado: " (str-from-int (calcular 5))))
      (println (concat "main helper: " (str-from-int (helper 5)))))))`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0600); err != nil {
		t.Fatal(err)
	}

	mod, diags := loader.Load(mainFile)
	if len(diags) > 0 {
		t.Fatalf("loader diags: %v", diags)
	}

	gen := codegen.New(mod)
	code := gen.Generate()

	if !strings.Contains(code, "func calcular(") {
		t.Errorf("codigo gerado nao contem 'calcular': %s", code)
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}

	binPath := filepath.Join(dir, "teste_app.exe")
	build := exec.Command("go", "build", "-o", binPath, source)
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("erro compilando Go: %v\n%s\nCodigo gerado:\n%s", err, out, code)
	}

	cmd := exec.Command(binPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("erro executando binario: %v (saida: %s)", err, string(out))
	}

	saida := string(out)
	if !strings.Contains(saida, "calculado: 15") {
		t.Errorf("saida esperada 'calculado: 15', obteve: %s", saida)
	}
	if !strings.Contains(saida, "main helper: 105") {
		t.Errorf("saida esperada 'main helper: 105', obteve: %s", saida)
	}
}
