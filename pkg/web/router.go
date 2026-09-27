package web

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type Request struct {
	Method, Path, Body string
	Header             http.Header
	Query              url.Values
}

type Response struct {
	Status      int
	Body        string
	ContentType string
	// Headers fica em ordem de chamada porque Set-Cookie pode repetir; um
	// map guardaria so o ultimo cookie.
	Headers []ResponseHeader
}

type ResponseHeader struct {
	Name, Value string
	Append      bool
}

// WithHeader devolve uma copia da resposta com o header acrescentado (substitui no envio final).
func (r Response) WithHeader(name, value string) Response {
	headers := make([]ResponseHeader, len(r.Headers), len(r.Headers)+1)
	copy(headers, r.Headers)
	r.Headers = append(headers, ResponseHeader{Name: name, Value: value, Append: false})
	return r
}

// AddHeader devolve uma copia da resposta acumulando o header (usa Header.Add no envio final).
func (r Response) AddHeader(name, value string) Response {
	headers := make([]ResponseHeader, len(r.Headers), len(r.Headers)+1)
	copy(headers, r.Headers)
	r.Headers = append(headers, ResponseHeader{Name: name, Value: value, Append: true})
	return r
}

func newRequest(req *http.Request, body string) Request {
	return Request{Method: req.Method, Path: req.URL.Path, Body: body, Header: req.Header, Query: req.URL.Query()}
}

// writeHeaders aplica os headers da rota depois do Content-Type padrao, para
// que a rota possa troca-lo (text/csv, por exemplo). Set-Cookie e Vary acumulam; os
// demais substituem a menos que adicionados com AddHeader. Um nome invalido ou um valor com
// quebra de linha e erro de programa, e falha alto em vez de sumir: o
// net/http descartaria o nome e trocaria a quebra por espaco em silencio.
func writeHeaders(w http.ResponseWriter, res Response) error {
	h := w.Header()
	h.Set("Content-Type", res.ContentType)
	for _, hd := range res.Headers {
		if !ValidHeaderName(hd.Name) {
			return fmt.Errorf("invalid response header name %q", hd.Name)
		}
		if strings.ContainsAny(hd.Value, "\r\n\x00") {
			return fmt.Errorf("response header %s has a line break or NUL in its value", hd.Name)
		}
		canonical := http.CanonicalHeaderKey(hd.Name)
		if hd.Append || canonical == "Set-Cookie" || canonical == "Vary" {
			h.Add(hd.Name, hd.Value)
		} else {
			h.Set(hd.Name, hd.Value)
		}
	}
	return nil
}

// ValidHeaderName segue o token da RFC 9110, secao 5.6.2. Exportada porque o
// cliente HTTP do runtime valida os headers de saida com a mesma regra.
func ValidHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			continue
		}
		if !strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c)) {
			return false
		}
	}
	return true
}

type Handler func(Request) Response
type Router struct{ mux *http.ServeMux }

func NewRouter() *Router { return &Router{mux: http.NewServeMux()} }
// MaxRequestBodySize define o limite máximo padrão do corpo da requisição (padrão: 32MB).
var MaxRequestBodySize int64 = 32 << 20

func (r *Router) Handle(method, path string, handler Handler) {
	r.mux.HandleFunc(method+" "+path, func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, req.Body, MaxRequestBodySize))
		if err != nil {
			http.Error(w, "Request body too large or unreadable", http.StatusRequestEntityTooLarge)
			return
		}
		res := handler(newRequest(req, string(body)))
		if res.Status < 100 || res.Status > 599 {
			http.Error(w, "Invalid response status", http.StatusInternalServerError)
			return
		}
		if err := writeHeaders(w, res); err != nil {
			for name := range w.Header() {
				w.Header().Del(name)
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(res.Status)
		_, _ = io.WriteString(w, res.Body)
	})
}

// HandleRaw registra um handler que recebe a requisicao crua. O WebSocket
// precisa disso: o upgrade toma posse da conexao TCP, e Handle acima ja teria
// lido o corpo e se comprometido a escrever uma Response.
func (r *Router) HandleRaw(method, path string, handler http.Handler) {
	r.mux.Handle(method+" "+path, handler)
}

func (r *Router) Handler(fallback http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		h, pattern := r.mux.Handler(q)
		if pattern != "" {
			h.ServeHTTP(w, q)
			return
		}
		// Preserve the mux's 405 response when another method owns this path.
		for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"} {
			copy := q.Clone(q.Context())
			copy.Method = method
			if _, p := r.mux.Handler(copy); p != "" {
				r.mux.ServeHTTP(w, q)
				return
			}
		}
		fallback.ServeHTTP(w, q)
	})
}
func JSONResponse(status int64, body string) Response {
	return Response{Status: int(status), Body: body, ContentType: "application/json; charset=utf-8"}
}

func RawResponse(status int64, contentType, body string) Response {
	return Response{Status: int(status), Body: body, ContentType: contentType}
}

var defaultRouter = NewRouter()
var routerMu sync.Mutex

func Register(method, path string, h Handler) {
	routerMu.Lock()
	defer routerMu.Unlock()
	defaultRouter.Handle(method, path, h)
}

// WSHandler e o laco completo de uma conexao: recebe a conexao ja aberta e a
// requisicao do handshake, e retorna quando a conexao termina.
type WSHandler func(*WSConn, Request)

// RegisterWS registra uma rota (ws-route ...). O handshake e sempre um GET,
// entao a rota convive com as rotas HTTP no mesmo mux — o que permite servir
// a pagina e o socket dela na mesma porta.
func RegisterWS(path string, h WSHandler) {
	routerMu.Lock()
	defer routerMu.Unlock()
	defaultRouter.HandleRaw(http.MethodGet, path, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		conn, err := Upgrade(w, req)
		if err != nil {
			http.Error(w, "WebSocket upgrade failed: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer conn.Close()
		h(conn, newRequest(req, ""))
	}))
}
func ServeHybrid(dir, port string) error {
	s, err := NewServer(ServerOptions{Dir: dir, Port: port})
	if err != nil {
		return err
	}
	return http.ListenAndServe(ListenAddr(port), defaultRouter.Handler(s))
}
