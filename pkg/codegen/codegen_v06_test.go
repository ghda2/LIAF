package codegen

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateV06LinearRoutes(t *testing.T) {
	code := generateExample(t, "task_api_v06")

	for _, want := range []string{
		`web.Register("GET", "/health"`,
		`web.Register("GET", "/tasks"`,
		`web.Register("POST", "/tasks"`,
	} {
		if !strings.Contains(code, want) {
			t.Errorf("codigo gerado nao registra a rota: %s", want)
		}
	}
}

func TestExecuteV06TaskAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("teste de integracao com go build ignorado em modo curto")
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte(generateExample(t, "task_api_v06")), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "public"), 0755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	bin := filepath.Join(dir, "task_api_v06.exe")
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

	// 1. GET /health
	res, err := client.Get(base + "/health")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(body), "healthy") {
		t.Fatalf("GET /health falhou: status %d body %s", res.StatusCode, string(body))
	}

	// 2. POST /tasks
	postRes, err := client.Post(base+"/tasks", "application/json", strings.NewReader(`{"title":"Comprar cafe"}`))
	if err != nil {
		t.Fatal(err)
	}
	postBody, _ := io.ReadAll(postRes.Body)
	postRes.Body.Close()
	if postRes.StatusCode != 201 {
		t.Fatalf("POST /tasks falhou: status %d body %s", postRes.StatusCode, string(postBody))
	}

	var created map[string]interface{}
	if err := json.Unmarshal(postBody, &created); err != nil {
		t.Fatalf("erro ao decodificar retorno do POST: %v", err)
	}
	if created["title"] != "Comprar cafe" {
		t.Fatalf("titulo retornado incorreto: %v", created["title"])
	}

	// 3. GET /tasks
	getRes, err := client.Get(base + "/tasks")
	if err != nil {
		t.Fatal(err)
	}
	getBody, _ := io.ReadAll(getRes.Body)
	getRes.Body.Close()
	if getRes.StatusCode != 200 || !strings.Contains(string(getBody), "Comprar cafe") {
		t.Fatalf("GET /tasks falhou: status %d body %s", getRes.StatusCode, string(getBody))
	}
}
