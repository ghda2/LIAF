package codegen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"liaf/pkg/checker"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func generateExample(t *testing.T, name string) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "examples", name+".liaf"))
	if err != nil {
		t.Fatal(err)
	}
	p := parser.New(lexer.New(string(data)), name)
	mod := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		t.Fatalf("parse: %+v", p.Diagnostics)
	}
	if _, diags := checker.Check(mod, name); len(diags) > 0 {
		t.Fatalf("check: %+v", diags)
	}
	return New(mod).Generate()
}

// TestGenerateV03Routes cobre o que o codegen precisa emitir para (route ...)
// sem pagar o custo de compilar: registro no roteador, extracao do parametro de
// path pelo indice correto e decodificacao automatica do corpo JSON.
func TestGenerateV03Routes(t *testing.T) {
	code := generateExample(t, "task_api_v03")

	for _, want := range []string{
		`web.Register("GET", "/health"`,
		`web.Register("GET", "/tasks"`,
		`web.Register("POST", "/tasks"`,
		`web.Register("GET", "/tasks/{id}"`,
		`web.Register("PUT", "/tasks/{id}"`,
		`web.Register("DELETE", "/tasks/{id}"`,
	} {
		if !strings.Contains(code, want) {
			t.Errorf("codigo gerado nao registra a rota: %s", want)
		}
	}

	// /tasks/{id} tem {id} no indice 1, nao no ultimo segmento por acaso.
	if !strings.Contains(code, "strconv.ParseInt(parts[1], 10, 64)") {
		t.Error("parametro de path deve ser extraido pelo indice do padrao")
	}
	if strings.Contains(code, "parts[len(parts)-1]") {
		t.Error("extracao pelo ultimo segmento quebra rotas como /users/{id}/tasks")
	}
	// O corpo JSON do PUT/POST e desserializado pelo wrapper, nao pelo usuario.
	if !strings.Contains(code, "json.Unmarshal([]byte(req.Body)") {
		t.Error("corpo da requisicao deve ser desserializado no wrapper da rota")
	}
	if !strings.Contains(code, `web.JSONResponse(400, `+"`"+`{"error":"Invalid JSON body"}`+"`"+`)`) {
		t.Error("corpo JSON invalido deve virar 400 automatico")
	}
}

// TestGenerateV03TryPropagation confirma que (try ...) vira desempacotamento
// plano com desvio para o bloco (on-err ...), sem piramide de match.
func TestGenerateV03TryPropagation(t *testing.T) {
	code := generateExample(t, "task_api_v03")

	if !strings.Contains(code, "_liaf_on_err := func(") {
		t.Error("bloco on-err deve virar uma funcao local")
	}
	if !strings.Contains(code, "return _liaf_on_err(") {
		t.Error("try deve desviar para o on-err quando o resultado falha")
	}
	if !strings.Contains(code, "_liaf_try_") {
		t.Error("try deve materializar o resultado numa variavel temporaria")
	}
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return fmt.Sprint(l.Addr().(*net.TCPAddr).Port)
}

// TestExecuteV03TaskAPI compila e executa examples/task_api_v03.liaf de verdade
// e exercita o ciclo completo da API pela rede.
func TestExecuteV03TaskAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("compila um binario; pulado em -short")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte(generateExample(t, "task_api_v03")), 0600); err != nil {
		t.Fatal(err)
	}
	// serve-hybrid exige uma raiz estatica ao lado do binario.
	if err := os.Mkdir(filepath.Join(dir, "public"), 0755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	bin := filepath.Join(dir, "task_api_v03.exe")
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

	// O servidor sobe de forma assincrona; espera a porta responder.
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

	do := func(method, path, body string) (int, string) {
		t.Helper()
		var reader io.Reader
		if body != "" {
			reader = strings.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, base+path, reader)
		if err != nil {
			t.Fatal(err)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		out, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		return res.StatusCode, strings.TrimSpace(string(out))
	}

	for _, tc := range []struct {
		name         string
		method, path string
		body         string
		status       int
		want         string
	}{
		{"health", "GET", "/health", "", 200, `{"status":"ok"}`},
		{"criar devolve 201", "POST", "/tasks", `{"title":"primeira"}`, 201, `{"id":1,"title":"primeira","done":false}`},
		{"criar segunda", "POST", "/tasks", `{"title":"segunda"}`, 201, `{"id":2,"title":"segunda","done":false}`},
		{"titulo vazio", "POST", "/tasks", `{"title":""}`, 400, `{"error":"A nonempty title is required"}`},
		{"json invalido", "POST", "/tasks", `{nao e json`, 400, `{"error":"Invalid JSON body"}`},
		{"buscar por id", "GET", "/tasks/1", "", 200, `{"id":1,"title":"primeira","done":false}`},
		{"id inexistente", "GET", "/tasks/999", "", 404, `{"error":"Task not found"}`},
		{"id nao numerico", "GET", "/tasks/abc", "", 400, `{"error":"ID must be positive"}`},
		{"atualizar", "PUT", "/tasks/1", `{"title":"editada","done":true}`, 200, `{"id":1,"title":"editada","done":true}`},
		{"remover", "DELETE", "/tasks/2", "", 200, `{"deleted":true}`},
		{"remover inexistente", "DELETE", "/tasks/2", "", 404, `{"error":"Task not found"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := do(tc.method, tc.path, tc.body)
			if status != tc.status {
				t.Errorf("status %d, esperado %d (corpo %s)", status, tc.status, body)
			}
			if body != tc.want {
				t.Errorf("corpo %s, esperado %s", body, tc.want)
			}
		})
	}

	// (list T) precisa sair como array JSON nativo, nao como {"Items":[...]}.
	t.Run("listas sao arrays JSON nativos", func(t *testing.T) {
		status, body := do("GET", "/tasks", "")
		if status != 200 {
			t.Fatalf("status %d: %s", status, body)
		}
		if strings.Contains(body, `"Items"`) {
			t.Fatalf("lista serializada com envelope Items: %s", body)
		}
		var tasks []struct {
			ID    int64  `json:"id"`
			Title string `json:"title"`
			Done  bool   `json:"done"`
		}
		if err := json.Unmarshal([]byte(body), &tasks); err != nil {
			t.Fatalf("resposta nao e um array JSON: %s", body)
		}
		if len(tasks) != 1 || tasks[0].ID != 1 || !tasks[0].Done {
			t.Fatalf("estado inesperado apos o ciclo completo: %s", body)
		}
	})

	// fs-write-atomic grava em .tmp e renomeia; nada deve sobrar para tras.
	t.Run("io atomico nao deixa arquivo temporario", func(t *testing.T) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if strings.Contains(e.Name(), ".tmp.") {
				t.Errorf("arquivo temporario deixado para tras: %s", e.Name())
			}
		}
	})
}
