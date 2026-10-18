package codegen_test

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"liaf/pkg/codegen"
	"liaf/pkg/loader"
)

func TestCodegenServerDrivenUI(t *testing.T) {
	dir := t.TempDir()

	mainFile := filepath.Join(dir, "main.liaf")
	mainContent := `(module main
  (import "std/html")

  (route GET "/componente/painel" () Response
    (let unsafe str "<script>alert('xss')</script>")
    (let seguro str (html-safe unsafe))
    (let card str (html-elem "div" "class=\"card\"" (concat "<h3>Painel de Controle</h3><p>" seguro "</p>")))
    (html-render 200 card))

  (fn main (params) (returns void) (effects fs io net)
    (println "Servidor Server-Driven UI rodando em :8089")
    (let _ (serve-hybrid "./public" "8089"))))`

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

	binPath := filepath.Join(dir, "ui_server.exe")
	build := exec.Command("go", "build", "-o", binPath, source)
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("erro compilando Go: %v\n%s\nCodigo gerado:\n%s", err, out, code)
	}

	// Executa o servidor em background
	cmd := exec.Command(binPath)
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		t.Fatalf("falha ao iniciar servidor: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
	}()

	// Aguarda o servidor subir
	time.Sleep(300 * time.Millisecond)

	// Faz requisição HTTP para a rota Server-Driven UI
	resp, err := http.Get("http://localhost:8089/componente/painel")
	if err != nil {
		t.Fatalf("erro ao requisitar componente HTML: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status esperado 200, obteve %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type esperado 'text/html', obteve: %s", ct)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	body := string(bodyBytes)
	if !strings.Contains(body, "<div class=\"card\">") {
		t.Errorf("HTML gerado sem tag div class card: %s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Errorf("HTML não escapou XSS corretamente: %s", body)
	}
	if strings.Contains(body, "<script>") {
		t.Errorf("HTML continha tag script não escapada: %s", body)
	}
}
