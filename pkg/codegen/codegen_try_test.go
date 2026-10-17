package codegen

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecuteNestedTry(t *testing.T) {
	if testing.Short() {
		t.Skip("teste de integracao com go build ignorado em modo curto")
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte(generateExample(t, "try_aninhado")), 0600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	bin := filepath.Join(dir, "try_aninhado.exe")
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, source)
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	out, err := exec.CommandContext(ctx, bin).CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}

	want := []string{
		"ok 42",
		`tratado: int-from-str: "abc" nao e um inteiro valido`,
		`padrao (int-from-str: "x" nao e um inteiro valido)`,
		"3 3",
		"[INFO] varios argumentos",
	}
	got := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(out), "\r\n", "\n")), "\n")
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("saida inesperada:\n%s\nesperado:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
