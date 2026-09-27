package codegen

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
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

	t.Run("basic auth", func(t *testing.T) {
		for _, tc := range []struct {
			name, header string
			status       int
			body         string
		}{
			{"sem header", "", 401, "missing Authorization header"},
			{"valido", "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:militar123")), 200, "admin:militar123"},
			{"senha errada", "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:errada")), 403, "forbidden: invalid credentials"},
			{"usuario errado", "Basic " + base64.StdEncoding.EncodeToString([]byte("root:militar123")), 403, "forbidden: invalid credentials"},
			{"sem separador", "Basic " + base64.StdEncoding.EncodeToString([]byte("adminpass")), 401, "malformed basic credentials: missing colon"},
			{"base64 corrompido", "Basic ???!@#", 401, "invalid base64 in basic auth"},
			{"vazio", "Basic   ", 401, "empty basic token"},
			{"outro esquema", "Bearer 123", 401, "expected Authorization: Basic <token>"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				headers := map[string]string{}
				if tc.header != "" {
					headers["Authorization"] = tc.header
				}
				res, body := do("GET", "/auth/basic", headers)
				if res.StatusCode != tc.status || !strings.Contains(body, `"v":"`+tc.body+`"`) {
					t.Errorf("recebido %d %s, esperado %d com %q", res.StatusCode, body, tc.status, tc.body)
				}
			})
		}
	})

	t.Run("api key", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			path    string
			headers map[string]string
			status  int
			body    string
		}{
			{"padrao valido", "/auth/api-key", map[string]string{"X-API-Key": "militar-super-secret-key-2026"}, 200, "militar-super-secret-key-2026"},
			{"padrao chave errada", "/auth/api-key", map[string]string{"X-API-Key": "chave-errada"}, 403, "forbidden: invalid key"},
			{"padrao sem header", "/auth/api-key", map[string]string{}, 401, "missing API key header"},
			{"padrao vazia", "/auth/api-key", map[string]string{"X-API-Key": "   "}, 401, "empty API key"},
			{"custom valido", "/auth/api-key-custom", map[string]string{"X-Custom-Auth": "custom-key-999"}, 200, "custom-key-999"},
			{"custom errado", "/auth/api-key-custom", map[string]string{"X-Custom-Auth": "errado"}, 403, "forbidden: invalid custom key"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				res, body := do("GET", tc.path, tc.headers)
				if res.StatusCode != tc.status || !strings.Contains(body, `"v":"`+tc.body+`"`) {
					t.Errorf("recebido %d %s, esperado %d com %q", res.StatusCode, body, tc.status, tc.body)
				}
			})
		}
	})

	t.Run("cookie auth", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			headers map[string]string
			status  int
			body    string
		}{
			{"sem cookie", map[string]string{}, 401, "missing Cookie header"},
			{"cookie unico", map[string]string{"Cookie": "session_id=sessao_123"}, 200, "sessao_123"},
			{"multiplos cookies", map[string]string{"Cookie": "theme=dark; session_id=sessao_abc; lang=pt"}, 200, "sessao_abc"},
			{"cookie com aspas", map[string]string{"Cookie": `session_id="com_aspas_limpas"`}, 200, "com_aspas_limpas"},
			{"cookie inexistente", map[string]string{"Cookie": "outra_coisa=123"}, 401, "cookie not found"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				res, body := do("GET", "/auth/cookie", tc.headers)
				if res.StatusCode != tc.status || !strings.Contains(body, `"v":"`+tc.body+`"`) {
					t.Errorf("recebido %d %s, esperado %d com %q", res.StatusCode, body, tc.status, tc.body)
				}
			})
		}
	})

	t.Run("set cookie format", func(t *testing.T) {
		res, _ := do("GET", "/auth/set-cookie", nil)
		setCookie := res.Header.Get("Set-Cookie")
		for _, expected := range []string{
			"session_id=segredo123",
			"Max-Age=3600",
			"Path=/app",
			"Domain=exemplo.com",
			"Secure",
			"HttpOnly",
			"SameSite=Lax",
		} {
			if !strings.Contains(setCookie, expected) {
				t.Errorf("Set-Cookie = %q, esperado conter %q", setCookie, expected)
			}
		}
	})

	t.Run("jwt bearer", func(t *testing.T) {
		makeJWT := func(secret, payload string) string {
			h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
			p := base64.RawURLEncoding.EncodeToString([]byte(payload))
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write([]byte(h + "." + p))
			sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
			return h + "." + p + "." + sig
		}

		secret := "chave-secreta-militar-32-bytes-ok!"
		nowSec := time.Now().Unix()

		// Valido
		validToken := makeJWT(secret, fmt.Sprintf(`{"sub":"agente007","exp":%d}`, nowSec+3600))
		res, body := do("GET", "/auth/jwt", map[string]string{"Authorization": "Bearer " + validToken})
		if res.StatusCode != 200 || !strings.Contains(body, "agente007") {
			t.Errorf("jwt valido falhou: %d %s", res.StatusCode, body)
		}

		// Assinatura adulterada
		badSigToken := validToken[:len(validToken)-4] + "XXXX"
		res, body = do("GET", "/auth/jwt", map[string]string{"Authorization": "Bearer " + badSigToken})
		if res.StatusCode != 401 || !strings.Contains(body, "jwt: invalid signature") {
			t.Errorf("jwt assinatura adulterada falhou: %d %s", res.StatusCode, body)
		}

		// Token vencido
		expiredToken := makeJWT(secret, fmt.Sprintf(`{"sub":"agente007","exp":%d}`, nowSec-3600))
		res, body = do("GET", "/auth/jwt", map[string]string{"Authorization": "Bearer " + expiredToken})
		if res.StatusCode != 401 || !strings.Contains(body, "jwt: token expired") {
			t.Errorf("jwt vencido falhou: %d %s", res.StatusCode, body)
		}
	})

	t.Run("password hash e verify", func(t *testing.T) {
		res, body := do("POST", "/auth/password", nil)
		if res.StatusCode != 200 || !strings.Contains(body, "password verify ok") {
			t.Errorf("password verify falhou: %d %s", res.StatusCode, body)
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
