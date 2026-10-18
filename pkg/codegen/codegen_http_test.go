package codegen

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestExecutePedidosAPI compila testdata/pedidos_api.liaf e confere pela rede
// o que request-header, request-query e response-set-header prometem:
// autenticacao por Bearer, isolamento entre restaurantes, filtro por query,
// CORS com preflight e Set-Cookie.
func TestExecutePedidosAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("compila um binario; pulado em -short")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	code := generateExample(t, "pedidos_api")
	for _, want := range []string{"rt.RequestHeader(", "rt.RequestQuery(", "rt.ResponseSetHeader("} {
		if !strings.Contains(code, want) {
			t.Errorf("codigo gerado nao chama %s", want)
		}
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "public"), 0755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	bin := filepath.Join(dir, "pedidos_api.exe")
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

	do := func(method, path, body string, headers map[string]string) (*http.Response, string) {
		t.Helper()
		var reader io.Reader
		if body != "" {
			reader = strings.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, base+path, reader)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, _ := io.ReadAll(res.Body)
		return res, string(data)
	}

	ready := false
	for i := 0; i < 100; i++ {
		if res, err := client.Get(base + "/api/pedidos"); err == nil {
			res.Body.Close()
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		t.Fatal("servidor nao respondeu")
	}

	type lista struct {
		Restaurante string `json:"restaurante"`
		Filtro      string `json:"filtro"`
		Pedidos     []struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
		} `json:"pedidos"`
	}
	decode := func(body string) lista {
		t.Helper()
		var l lista
		if err := json.Unmarshal([]byte(body), &l); err != nil {
			t.Fatalf("json: %v\n%s", err, body)
		}
		return l
	}
	bearer := func(token string) map[string]string {
		return map[string]string{"Authorization": "Bearer " + token}
	}

	t.Run("sem login", func(t *testing.T) {
		res, body := do("GET", "/api/pedidos", "", nil)
		if res.StatusCode != 401 {
			t.Fatalf("status %d: %s", res.StatusCode, body)
		}
		if got := res.Header.Get("WWW-Authenticate"); got != "Bearer" {
			t.Errorf("WWW-Authenticate: %q", got)
		}
		// Os headers comuns valem tambem para respostas de erro.
		if got := res.Header.Get("Cache-Control"); got != "no-store" {
			t.Errorf("Cache-Control: %q", got)
		}
	})

	t.Run("token invalido ou esquema errado", func(t *testing.T) {
		for _, h := range []map[string]string{bearer("tok-nao-existe"), {"Authorization": "Basic dXNlcjpzZW5oYQ=="}} {
			if res, body := do("GET", "/api/pedidos", "", h); res.StatusCode != 401 {
				t.Errorf("%v: status %d: %s", h, res.StatusCode, body)
			}
		}
	})

	t.Run("cada restaurante ve so os seus pedidos", func(t *testing.T) {
		res, body := do("GET", "/api/pedidos", "", bearer("tok-pizzaria"))
		if res.StatusCode != 200 {
			t.Fatalf("status %d: %s", res.StatusCode, body)
		}
		if l := decode(body); l.Restaurante != "pizzaria" || len(l.Pedidos) != 3 || l.Filtro != "todos" {
			t.Errorf("pizzaria: %+v", l)
		}
		// Nome do header em minusculas: HTTP nao diferencia.
		_, body = do("GET", "/api/pedidos", "", map[string]string{"authorization": "Bearer tok-sushi"})
		if l := decode(body); l.Restaurante != "sushi" || len(l.Pedidos) != 1 || l.Pedidos[0].ID != 10 {
			t.Errorf("sushi: %+v", l)
		}
	})

	t.Run("filtro pela query string", func(t *testing.T) {
		_, body := do("GET", "/api/pedidos?status=aberto", "", bearer("tok-pizzaria"))
		l := decode(body)
		if l.Filtro != "aberto" || len(l.Pedidos) != 2 {
			t.Fatalf("%+v", l)
		}
		for _, p := range l.Pedidos {
			if p.Status != "aberto" {
				t.Errorf("pedido fora do filtro: %+v", p)
			}
		}
	})

	t.Run("preflight de CORS", func(t *testing.T) {
		res, _ := do("OPTIONS", "/api/pedidos", "", map[string]string{
			"Origin":                        "https://painel.exemplo.com",
			"Access-Control-Request-Method": "GET",
		})
		if res.StatusCode != 204 {
			t.Fatalf("status %d", res.StatusCode)
		}
		for name, want := range map[string]string{
			"Access-Control-Allow-Origin":  "https://painel.exemplo.com",
			"Access-Control-Allow-Methods": "GET, OPTIONS",
			"Access-Control-Allow-Headers": "Authorization",
			"Access-Control-Max-Age":       "600",
			"Vary":                         "Origin",
		} {
			if got := res.Header.Get(name); got != want {
				t.Errorf("%s: %q", name, got)
			}
		}

		// Origem fora da lista: sem permissao nenhuma.
		res, _ = do("OPTIONS", "/api/pedidos", "", map[string]string{"Origin": "https://evil.exemplo"})
		if res.StatusCode != 403 || res.Header.Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("preflight de origem desconhecida: %d %q", res.StatusCode, res.Header.Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("CORS so ecoa origem da lista", func(t *testing.T) {
		for origin, want := range map[string]string{
			"https://painel.exemplo.com": "https://painel.exemplo.com",
			"https://evil.exemplo":       "",
			"":                           "",
		} {
			h := bearer("tok-pizzaria")
			if origin != "" {
				h["Origin"] = origin
			}
			res, _ := do("GET", "/api/pedidos", "", h)
			if res.StatusCode != 200 {
				t.Fatalf("%q: status %d", origin, res.StatusCode)
			}
			if got := res.Header.Get("Access-Control-Allow-Origin"); got != want {
				t.Errorf("origem %q: Allow-Origin %q, esperado %q", origin, got, want)
			}
		}
	})

	address := "127.0.0.1:" + port

	t.Run("painel so recebe os avisos do proprio restaurante", func(t *testing.T) {
		pizzaria := dialChat(t, address, "/ws/painel?token=tok-pizzaria")
		if op, text := pizzaria.nextFrame(); op != 0x1 || text != "conectado:pizzaria" {
			t.Fatalf("pizzaria: op %x %q", op, text)
		}
		sushi := dialChat(t, address, "/ws/painel?token=tok-sushi")
		if op, text := sushi.nextFrame(); op != 0x1 || text != "conectado:sushi" {
			t.Fatalf("sushi: op %x %q", op, text)
		}

		res, body := do("POST", "/api/avisos", "pedido 42", bearer("tok-pizzaria"))
		if res.StatusCode != 200 || body != `{"entregues":1}` {
			t.Fatalf("aviso pizzaria: %d %s", res.StatusCode, body)
		}
		if _, text := pizzaria.nextFrame(); text != "pedido 42" {
			t.Errorf("pizzaria recebeu %q", text)
		}
		// Se o aviso da pizzaria tivesse vazado, seria o proximo frame do
		// sushi, antes do aviso dele.
		do("POST", "/api/avisos", "pedido 7", bearer("tok-sushi"))
		if _, text := sushi.nextFrame(); text != "pedido 7" {
			t.Errorf("sushi recebeu %q: aviso de outro restaurante vazou", text)
		}

		if res, _ := do("POST", "/api/avisos", "x", nil); res.StatusCode != 401 {
			t.Errorf("aviso sem login: %d", res.StatusCode)
		}
	})

	t.Run("painel sem token valido e fechado com 4401", func(t *testing.T) {
		for _, path := range []string{"/ws/painel?token=tok-errado", "/ws/painel"} {
			c := dialChat(t, address, path)
			op, payload := c.nextFrame()
			if op != 0x8 || len(payload) < 2 {
				t.Fatalf("%s: esperado close, veio op %x %q", path, op, payload)
			}
			if code := int(payload[0])<<8 | int(payload[1]); code != 4401 {
				t.Errorf("%s: codigo de fechamento %d", path, code)
			}
		}
	})

	t.Run("login define cookie", func(t *testing.T) {
		res, body := do("POST", "/api/login", `{"token":"tok-sushi"}`, nil)
		if res.StatusCode != 200 {
			t.Fatalf("status %d: %s", res.StatusCode, body)
		}
		cookie := res.Header.Get("Set-Cookie")
		if !strings.HasPrefix(cookie, "sessao=tok-sushi") || !strings.Contains(cookie, "HttpOnly") {
			t.Errorf("Set-Cookie: %q", cookie)
		}
		res, _ = do("POST", "/api/login", `{"token":"errado"}`, nil)
		if res.StatusCode != 401 || res.Header.Get("Set-Cookie") != "" {
			t.Errorf("login invalido: status %d, cookie %q", res.StatusCode, res.Header.Get("Set-Cookie"))
		}
	})
}

// nextFrame devolve o proximo frame de texto ou de fechamento, pulando ping
// e pong. Diferente de nextEvent, nao exige JSON e deixa ver o close.
func (c *testWSClient) nextFrame() (byte, string) {
	c.t.Helper()
	for {
		c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		header := make([]byte, 2)
		if _, err := io.ReadFull(c.r, header); err != nil {
			c.t.Fatalf("leitura: %v", err)
		}
		size := int(header[1] & 0x7f)
		if size == 126 {
			extended := make([]byte, 2)
			io.ReadFull(c.r, extended)
			size = int(extended[0])<<8 | int(extended[1])
		}
		payload := make([]byte, size)
		if _, err := io.ReadFull(c.r, payload); err != nil {
			c.t.Fatalf("payload: %v", err)
		}
		if op := header[0] & 0x0f; op == 0x1 || op == 0x8 {
			return op, string(payload)
		}
	}
}

// TestExecuteCobrancaPix compila testdata/cobranca_pix.liaf e o roda contra
// um PSP falso: confere o que o http-fetch envia (metodo, token, chave de
// idempotencia, JSON) e como o programa trata sucesso e erro do provedor.
func TestExecuteCobrancaPix(t *testing.T) {
	if testing.Short() {
		t.Skip("compila um binario; pulado em -short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	type recebida struct {
		method, path, auth, key, contentType, body string
	}
	var mu sync.Mutex
	var chamadas []recebida
	falhar := false
	psp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		chamadas = append(chamadas, recebida{r.Method, r.URL.Path, r.Header.Get("Authorization"),
			r.Header.Get("Idempotency-Key"), r.Header.Get("Content-Type"), string(b)})
		f := falhar
		mu.Unlock()
		if f {
			w.WriteHeader(422)
			io.WriteString(w, `{"error":"valor minimo e 100"}`)
			return
		}
		w.WriteHeader(201)
		io.WriteString(w, `{"txid":"tx-123","copia_e_cola":"00020126...6304ABCD"}`)
	}))
	defer psp.Close()

	dir := t.TempDir()
	bin := buildExample(t, ctx, "cobranca_pix", dir)
	run := func(valor string) (string, int) {
		t.Helper()
		cmd := exec.CommandContext(ctx, bin, valor)
		cmd.Env = append(os.Environ(), "PSP_URL="+psp.URL, "PSP_TOKEN=tok-secreto")
		out, err := cmd.Output()
		code := 0
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		return string(out), code
	}

	out, code := run("1500")
	if code != 0 || !strings.Contains(out, "txid: tx-123") || !strings.Contains(out, "pix: 00020126...6304ABCD") {
		t.Fatalf("sucesso: exit %d\n%s", code, out)
	}
	c := chamadas[0]
	if c.method != "POST" || c.path != "/v1/cobrancas" || c.auth != "Bearer tok-secreto" || c.contentType != "application/json" {
		t.Errorf("requisicao: %+v", c)
	}
	if c.body != `{"valor":1500,"descricao":"Pedido LIAF"}` {
		t.Errorf("corpo: %s", c.body)
	}
	if len(c.key) != 43 {
		t.Errorf("Idempotency-Key devia ser um random-token: %q", c.key)
	}

	mu.Lock()
	falhar = true
	mu.Unlock()
	out, code = run("50")
	if code != 1 || !strings.Contains(out, "PSP respondeu 422") || !strings.Contains(out, "valor minimo e 100") {
		t.Fatalf("erro do PSP: exit %d\n%s", code, out)
	}
	// Cada tentativa leva uma chave nova.
	if chamadas[1].key == c.key {
		t.Error("Idempotency-Key repetida entre cobrancas diferentes")
	}
}
