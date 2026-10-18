package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Criterio da issue #032: o executavel liafc, copiado sozinho para uma pasta
// qualquer, compila e roda um .liaf que tambem esta fora do repositorio, sem
// go.mod, sem replace e sem rede (GOPROXY=off: as dependencias de terceiros
// saem do cache de modulos que o proprio go test ja usou).
func TestLiafcBuildsOutsideRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("compila o liafc e um programa")
	}
	exe := "liafc"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	liafc := filepath.Join(t.TempDir(), exe)
	if out, err := exec.Command("go", "build", "-o", liafc, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build liafc: %v\n%s", err, out)
	}

	work := t.TempDir()
	src := `(module fora
  (struct Ponto (x int) (y int))
  (fn main () void (effects io fs)
    (on-err e (println e))
    (let p Ponto (new Ponto 3 4))
    (println (fmt "{} {}" p.x p.y))
    (println (try (fs-read-file "dado.txt")))))
`
	if err := os.WriteFile(filepath.Join(work, "app.liaf"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "dado.txt"), []byte("relativo"), 0o644); err != nil {
		t.Fatal(err)
	}

	env := append(os.Environ(), "LIAF_CACHE="+filepath.Join(t.TempDir(), "cache"), "GOPROXY=off", "GOFLAGS=-mod=readonly")
	liafcCmd := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(liafc, args...)
		cmd.Dir = work
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("liafc %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return string(out)
	}

	// run executa na pasta do usuario: o caminho relativo resolve em work.
	if got := liafcCmd("run", "app.liaf"); !strings.Contains(got, "3 4\nrelativo") {
		t.Fatalf("liafc run: %q", got)
	}

	liafcCmd("build", "app.liaf", "-o", "app")
	bin := filepath.Join(work, "app")
	if runtime.GOOS == "windows" {
		bin += ".exe" // liafc acrescenta, como o go build
	}
	run := exec.Command(bin)
	run.Dir = work
	out, err := run.CombinedOutput()
	if err != nil || !strings.Contains(strings.ReplaceAll(string(out), "\r\n", "\n"), "3 4\nrelativo") {
		t.Fatalf("binario gerado: %v\n%s", err, out)
	}
}
