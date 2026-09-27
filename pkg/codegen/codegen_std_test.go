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

// TestExecuteStdWeb compila testdata/std_web.liaf e confere pela rede o que
// std/auth e std/cors prometem. Nao ha como montar um Request na LIAF, entao
// esses modulos so podem ser testados com um servidor de verdade.
func TestExecuteStdWeb(t *testing.T) {
	if testing.Short() {
		t.Skip("compila um binario; pulado em -short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "public"), 0755); err != nil {
		t.Fatal(err)
	}
	bin := buildExample(t, ctx, "std_web", dir)

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
	do := func(method, path string, headers map[string]string) (*http.Response, string) {
		t.Helper()
		req, err := http.NewRequestWithContext(ctx, method, base+path, nil)
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
		if res, err := client.Get(base + "/token"); err == nil {
			res.Body.Close()
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		t.Fatal("servidor nao respondeu")
	}

	t.Run("auth", func(t *testing.T) {
		for _, tc := range []struct {
			name, header string
			status       int
			body         string
		}{
			{"sem header", "", 401, "missing Authorization header"},
			{"bearer", "Bearer abc.def", 200, "abc.def"},
			{"minusculo", "bearer abc", 200, "abc"},
			{"maiusculo", "BEARER abc", 200, "abc"},
			{"espacos", "Bearer   abc  ", 200, "abc"},
			{"basic", "Basic dXNlcjpwYXNz", 401, "expected Authorization: Bearer <token>"},
			{"curto", "Bear", 401, "expected Authorization: Bearer <token>"},
			{"sem espaco", "Bearerabc", 401, "expected Authorization: Bearer <token>"},
			{"vazio", "Bearer    ", 401, "empty bearer token"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				headers := map[string]string{}
				if tc.header != "" {
					headers["Authorization"] = tc.header
				}
				res, body := do("GET", "/token", headers)
				if res.StatusCode != tc.status || !strings.Contains(body, `"v":"`+tc.body+`"`) {
					t.Errorf("recebido %d %s, esperado %d com %q", res.StatusCode, body, tc.status, tc.body)
				}
			})
		}
	})

	const permitida = "https://painel.exemplo.com"
	t.Run("cors permitida", func(t *testing.T) {
		res, _ := do("GET", "/cors", map[string]string{"Origin": permitida})
		if got := res.Header.Get("Access-Control-Allow-Origin"); got != permitida {
			t.Errorf("Allow-Origin = %q", got)
		}
		if got := res.Header.Get("Vary"); got != "Origin" {
			t.Errorf("Vary = %q", got)
		}
		if got := res.Header.Get("Access-Control-Allow-Credentials"); got != "" {
			t.Errorf("sem cookie nao deve haver Allow-Credentials, veio %q", got)
		}
	})
	t.Run("cors recusada", func(t *testing.T) {
		for origem, motivo := range map[string]string{
			"https://evil.exemplo.com": "cors: origin not allowed: https://evil.exemplo.com",
			"":                         "cors: request without Origin header",
		} {
			headers := map[string]string{}
			if origem != "" {
				headers["Origin"] = origem
			}
			res, _ := do("GET", "/cors", headers)
			if got := res.Header.Get("Access-Control-Allow-Origin"); got != "" {
				t.Errorf("origem %q recebeu Allow-Origin %q", origem, got)
			}
			if got := res.Header.Get("X-Cors-Erro"); got != motivo {
				t.Errorf("origem %q: motivo %q, esperado %q", origem, got, motivo)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		res, _ := do("OPTIONS", "/cors", map[string]string{"Origin": permitida, "Access-Control-Request-Method": "GET"})
		want := map[string]string{
			"Access-Control-Allow-Origin":  permitida,
			"Access-Control-Allow-Methods": "GET, OPTIONS",
			"Access-Control-Allow-Headers": "Authorization",
			"Access-Control-Max-Age":       "600",
		}
		if res.StatusCode != 204 {
			t.Errorf("status %d, esperado 204", res.StatusCode)
		}
		for k, v := range want {
			if got := res.Header.Get(k); got != v {
				t.Errorf("%s = %q, esperado %q", k, got, v)
			}
		}
		if got := res.Header.Get("Access-Control-Allow-Credentials"); got != "" {
			t.Errorf("Allow-Credentials = %q, esperado ausente", got)
		}
		res, _ = do("OPTIONS", "/cors", map[string]string{"Origin": "https://evil.exemplo.com"})
		if res.StatusCode != 403 || res.Header.Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("preflight de origem fora da lista: %d %v", res.StatusCode, res.Header)
		}
	})
	t.Run("credenciais", func(t *testing.T) {
		res, _ := do("GET", "/cors-cookie", map[string]string{"Origin": permitida})
		if res.Header.Get("Access-Control-Allow-Origin") != permitida || res.Header.Get("Access-Control-Allow-Credentials") != "true" {
			t.Errorf("headers: %v", res.Header)
		}
		res, _ = do("OPTIONS", "/cors-cookie", map[string]string{"Origin": permitida})
		if res.StatusCode != 204 || res.Header.Get("Access-Control-Allow-Credentials") != "true" || res.Header.Get("Access-Control-Allow-Methods") != "POST" {
			t.Errorf("preflight: %d %v", res.StatusCode, res.Header)
		}
	})
}
