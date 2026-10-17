package codegen

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateMiddlewareChain(t *testing.T) {
	code := generateExample(t, "middleware_api")

	first := strings.Index(code, sanitizeIdent("exigir_token")+"(req); !mw.OK")
	second := strings.Index(code, sanitizeIdent("marcar")+"(req); !mw.OK")
	if first < 0 || second < 0 {
		t.Fatalf("handler nao chama os middlewares do with(...):\n%s", code)
	}
	if first > second {
		t.Error("middlewares gerados fora da ordem do with(...)")
	}
	if strings.Count(code, "!mw.OK") != 2 {
		t.Error("uma rota sem with(...) ganhou chamada de middleware")
	}
}

func TestExecuteMiddlewareAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("teste de integracao com go build ignorado em modo curto")
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte(generateExample(t, "middleware_api")), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "public"), 0755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	bin := filepath.Join(dir, "middleware_api.exe")
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, source)
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	port := freePort(t)
	server := exec.CommandContext(ctx, bin, port)
	server.Dir = dir
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = server.Process.Kill()
		_, _ = server.Process.Wait()
	}()

	base := "http://127.0.0.1:" + port
	client := &http.Client{Timeout: 5 * time.Second}

	ready := false
	for i := 0; i < 100; i++ {
		if res, err := client.Get(base + "/health"); err == nil {
			res.Body.Close()
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		t.Fatal("servidor nao respondeu em /health")
	}

	get := func(path, auth string) (int, string) {
		t.Helper()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(body)
	}

	// Rejeicao: o middleware encerra a cadeia e a rota nao roda.
	for _, auth := range []string{"", "Bearer errado"} {
		status, body := get("/painel/7", auth)
		if status != 401 || !strings.Contains(body, "token obrigatorio") {
			t.Errorf("Authorization %q: esperado 401, obteve %d %s", auth, status, body)
		}
	}

	// Sucesso: os dois middlewares rodam em ordem e o contexto chega a rota,
	// junto do parametro de path.
	status, body := get("/painel/7", "Bearer segredo")
	if status != 200 {
		t.Fatalf("com token: esperado 200, obteve %d %s", status, body)
	}
	for _, want := range []string{`"id":7`, `"usuario":"ana"`, `"visto":"ana"`} {
		if !strings.Contains(body, want) {
			t.Errorf("com token: resposta %s sem %s", body, want)
		}
	}

	// Um header enviado pelo cliente nao sobrevive ao middleware que o define.
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/painel/1", nil)
	req.Header.Set("Authorization", "Bearer segredo")
	req.Header.Set("X-Usuario", "intruso")
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	spoofed, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if strings.Contains(string(spoofed), "intruso") {
		t.Errorf("header forjado pelo cliente chegou a rota: %s", spoofed)
	}
}
