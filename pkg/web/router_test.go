package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func serveOnce(t *testing.T, method, path string, h Handler, setup func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	r := NewRouter()
	r.Handle(method, "/api/pedidos", h)
	req := httptest.NewRequest(method, path, nil)
	if setup != nil {
		setup(req)
	}
	w := httptest.NewRecorder()
	r.Handler(http.NotFoundHandler()).ServeHTTP(w, req)
	return w
}

func TestRouterPassesHeadersAndQuery(t *testing.T) {
	var got Request
	serveOnce(t, "GET", "/api/pedidos?status=aberto&status=pronto&vazio=", func(req Request) Response {
		got = req
		return JSONResponse(200, "{}")
	}, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer abc")
	})

	if got.Path != "/api/pedidos" {
		t.Errorf("a query string vazou para o path: %q", got.Path)
	}
	// HTTP nao diferencia maiusculas no nome do header.
	if v := got.Header.Get("authorization"); v != "Bearer abc" {
		t.Errorf("Authorization: %q", v)
	}
	if v := got.Query["status"]; len(v) != 2 || v[0] != "aberto" {
		t.Errorf("status: %v", v)
	}
	if _, ok := got.Query["vazio"]; !ok {
		t.Error("parametro presente e vazio sumiu da query")
	}
}

func TestRouterWritesResponseHeaders(t *testing.T) {
	w := serveOnce(t, "GET", "/api/pedidos", func(Request) Response {
		return JSONResponse(200, "a;b").
			WithHeader("Cache-Control", "no-cache").
			WithHeader("Cache-Control", "no-store").
			WithHeader("Set-Cookie", "sessao=1; HttpOnly").
			WithHeader("set-cookie", "tema=escuro").
			WithHeader("Content-Type", "text/csv")
	}, nil)

	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	if v := w.Header().Values("Cache-Control"); len(v) != 1 || v[0] != "no-store" {
		t.Errorf("header repetido devia ficar com o ultimo valor: %v", v)
	}
	// Set-Cookie acumula: cada cookie e um header proprio.
	if v := w.Header().Values("Set-Cookie"); len(v) != 2 {
		t.Errorf("Set-Cookie: %v", v)
	}
	if v := w.Header().Get("Content-Type"); v != "text/csv" {
		t.Errorf("a rota nao conseguiu trocar o Content-Type: %q", v)
	}
}

func TestRouterRejectsInvalidResponseHeaders(t *testing.T) {
	for _, tc := range []struct{ name, header, value string }{
		{"nome com espaco", "Meu Header", "x"},
		{"nome vazio", "", "x"},
		{"injecao de header", "X-Nome", "a\r\nSet-Cookie: admin=1"},
		{"NUL no valor", "X-Nome", "a\x00b"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := serveOnce(t, "GET", "/api/pedidos", func(Request) Response {
				return JSONResponse(200, `{"segredo":true}`).
					WithHeader("Set-Cookie", "sessao=1").
					WithHeader(tc.header, tc.value)
			}, nil)
			if w.Code != http.StatusInternalServerError {
				t.Fatalf("status %d, esperado 500", w.Code)
			}
			// Nem o corpo nem os headers validos anteriores podem sair
			// junto com a resposta de erro.
			if len(w.Header().Values("Set-Cookie")) != 0 {
				t.Error("header valido vazou na resposta de erro")
			}
			if w.Body.String() == `{"segredo":true}` {
				t.Error("corpo da rota vazou na resposta de erro")
			}
		})
	}
}

// A mesma resposta base reaproveitada em dois ramos nao pode compartilhar a
// lista de headers: append sobre o mesmo array faria um ramo ver o header do
// outro.
func TestWithHeaderDoesNotAliasBase(t *testing.T) {
	base := JSONResponse(200, "{}").WithHeader("A", "1")
	left := base.WithHeader("B", "left")
	right := base.WithHeader("B", "right")
	if len(base.Headers) != 1 {
		t.Errorf("base alterada: %v", base.Headers)
	}
	if left.Headers[1].Value != "left" || right.Headers[1].Value != "right" {
		t.Errorf("ramos se contaminaram: %v %v", left.Headers, right.Headers)
	}
}

func TestListenAddr(t *testing.T) {
	for _, tc := range []struct{ host, port, want string }{
		{"", "8080", ":8080"}, // sem LIAF_HOST: todas as interfaces, como antes
		{"", ":8080", ":8080"},
		{"127.0.0.1", "8080", "127.0.0.1:8080"},
		{" 127.0.0.1 ", ":9000", "127.0.0.1:9000"},
		{"::1", "8080", "[::1]:8080"},
	} {
		t.Setenv("LIAF_HOST", tc.host)
		if got := ListenAddr(tc.port); got != tc.want {
			t.Errorf("host %q porta %q: %q, esperado %q", tc.host, tc.port, got, tc.want)
		}
	}
}
