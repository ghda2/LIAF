package runtime

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func strs(items ...string) *List[string] { return &List[string]{Items: items} }

func TestHTTPFetchSendsRequest(t *testing.T) {
	var got *http.Request
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("X-Request-Id", "abc")
		w.WriteHeader(201)
		io.WriteString(w, `{"id":7,"nome":"ação"}`)
	}))
	defer srv.Close()

	r := HTTPFetch("POST", srv.URL+"/pix?x=1", strs("Authorization: Bearer tok", "Idempotency-Key:  k-1 "), `{"valor":10}`)
	if !r.OK {
		t.Fatal(r.Error)
	}
	if got.Method != "POST" || got.URL.Path != "/pix" || got.URL.Query().Get("x") != "1" {
		t.Errorf("requisicao: %s %s", got.Method, got.URL)
	}
	if got.Header.Get("Authorization") != "Bearer tok" || got.Header.Get("Idempotency-Key") != "k-1" {
		t.Errorf("headers: %v", got.Header)
	}
	// Corpo sem Content-Type declarado vai como JSON.
	if got.Header.Get("Content-Type") != "application/json" || gotBody != `{"valor":10}` {
		t.Errorf("corpo: %q %q", got.Header.Get("Content-Type"), gotBody)
	}
	if ReplyStatus(r.Value) != 201 || ReplyBody(r.Value) != `{"id":7,"nome":"ação"}` {
		t.Errorf("resposta: %d %s", ReplyStatus(r.Value), ReplyBody(r.Value))
	}
	if h := ReplyHeader(r.Value, "x-request-id"); !h.OK || h.Value != "abc" {
		t.Errorf("header da resposta: %+v", h)
	}
	if h := ReplyHeader(r.Value, "X-Ausente"); h.OK {
		t.Errorf("header ausente devolveu ok: %+v", h)
	}
}

func TestHTTPFetchExplicitContentTypeWins(t *testing.T) {
	var ct string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ct = r.Header.Get("Content-Type") }))
	defer srv.Close()
	HTTPFetch("POST", srv.URL, strs("Content-Type: application/x-www-form-urlencoded"), "a=1")
	if ct != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type: %q", ct)
	}
}

// Como no fetch do JavaScript: 4xx e 5xx sao respostas, nao erros. O corpo
// costuma explicar o problema e o programa precisa le-lo.
func TestHTTPFetchErrorStatusIsAReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		io.WriteString(w, `{"error":"cpf invalido"}`)
	}))
	defer srv.Close()
	r := HTTPFetch("GET", srv.URL, strs(), "")
	if !r.OK || ReplyStatus(r.Value) != 422 || !strings.Contains(ReplyBody(r.Value), "cpf invalido") {
		t.Fatalf("%+v", r)
	}
}

func TestHTTPFetchRejectsBadInput(t *testing.T) {
	for name, tc := range map[string]struct {
		method, url string
		headers     *List[string]
	}{
		"metodo desconhecido":     {"POTS", "http://127.0.0.1/", strs()},
		"sem esquema":             {"GET", "127.0.0.1/x", strs()},
		"esquema file":            {"GET", "file:///etc/passwd", strs()},
		"sem host":                {"GET", "http:///x", strs()},
		"header sem dois-pontos":  {"GET", "http://127.0.0.1/", strs("Authorization Bearer x")},
		"nome de header invalido": {"GET", "http://127.0.0.1/", strs("Meu Header: x")},
		"injecao de header":       {"GET", "http://127.0.0.1/", strs("X-A: 1\r\nX-B: 2")},
	} {
		if r := HTTPFetch(tc.method, tc.url, tc.headers, ""); r.OK || !strings.HasPrefix(r.Error, "http-fetch: ") {
			t.Errorf("%s: %+v", name, r)
		}
	}
}

func TestHTTPFetchTransportErrorsHideTheQuery(t *testing.T) {
	// Uma porta sem ninguem escutando.
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := l.Addr().String()
	l.Close()
	r := HTTPFetch("GET", "http://"+addr+"/x?access_token=SEGREDO", strs(), "")
	if r.OK {
		t.Fatal("conexao recusada devolveu ok")
	}
	if strings.Contains(r.Error, "SEGREDO") || !strings.Contains(r.Error, addr) {
		t.Errorf("mensagem devia ter o host e nao a query: %s", r.Error)
	}
}

func TestHTTPFetchTimeout(t *testing.T) {
	old := fetchTimeout
	fetchTimeout = 200 * time.Millisecond
	defer func() { fetchTimeout = old }()

	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-release }))
	defer srv.Close()
	defer close(release)

	start := time.Now()
	r := HTTPFetch("GET", srv.URL, strs(), "")
	if r.OK || !strings.Contains(r.Error, "timeout") {
		t.Fatalf("%+v", r)
	}
	if time.Since(start) > 5*time.Second {
		t.Errorf("timeout nao foi respeitado: %s", time.Since(start))
	}
}

func TestHTTPFetchBodyLimits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/grande":
			io.WriteString(w, strings.Repeat("a", fetchMaxBody+1))
		case "/binario":
			w.Header().Set("Content-Type", "image/png")
			w.Write([]byte{0x89, 'P', 'N', 'G', 0xff, 0xfe})
		}
	}))
	defer srv.Close()
	if r := HTTPFetch("GET", srv.URL+"/grande", strs(), ""); r.OK || !strings.Contains(r.Error, "larger than") {
		t.Errorf("corpo acima do limite: %+v", r.Error)
	}
	if r := HTTPFetch("GET", srv.URL+"/binario", strs(), ""); r.OK || !strings.Contains(r.Error, "not UTF-8") {
		t.Errorf("corpo binario: %+v", r.Error)
	}
}
