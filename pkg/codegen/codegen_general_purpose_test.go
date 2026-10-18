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

func TestCodegenGeneralPurposeStandaloneCLI(t *testing.T) {
	dir := t.TempDir()

	mainFile := filepath.Join(dir, "main.liaf")
	mainContent := `(module main
  (import "std/cli")
  (import "std/fs")
  (import "std/csv")

  (fn main (params) (returns void) (effects io fs)
    (body
      (let args (list str) (cli-args-list))
      (println (concat "args count: " (str-from-int (cli-args-len))))

      ;; Testa CSV
      (let csv-raw str "id,produto,preco\n1,cafe,10\n2,leite,5")
      (match (csv-decode csv-raw)
        (ok linhas (println (concat "csv linhas: " (str-from-int (list-len linhas)))))
        (err e (println (concat "csv erro: " e))))

      ;; Testa FS
      (let subDir str "tmp_test_dir")
      (match (fs-make-dir subDir)
        (ok _ (println "dir criado com sucesso"))
        (err e (println (concat "erro dir: " e))))

      (if (fs-is-directory subDir)
        (then (println "verificado: e diretorio")))

      (match (fs-delete subDir)
        (ok _ (println "dir removido com sucesso"))
        (err e (println (concat "erro rm: " e)))))))`

	if err := os.WriteFile(mainFile, []byte(mainContent), 0600); err != nil {
		t.Fatal(err)
	}

	mod, diags := loader.Load(mainFile)
	if len(diags) > 0 {
		t.Fatalf("loader diags: %v", diags)
	}

	gen := codegen.New(mod)
	code := gen.Generate()

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}

	binPath := filepath.Join(dir, "cli_app.exe")
	build := exec.Command("go", "build", "-o", binPath, source)
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("erro compilando Go: %v\n%s\nCodigo gerado:\n%s", err, out, code)
	}

	cmd := exec.Command(binPath, "--flag1", "valor1")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("erro executando binario: %v (saida: %s)", err, string(out))
	}

	saida := string(out)
	for _, want := range []string{
		"args count: 2",
		"csv linhas: 3",
		"dir criado com sucesso",
		"verificado: e diretorio",
		"dir removido com sucesso",
	} {
		if !strings.Contains(saida, want) {
			t.Errorf("saida esperada nao encontrada: %q (saida completa:\n%s)", want, saida)
		}
	}
}
