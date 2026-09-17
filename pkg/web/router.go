package web

import (
	"io"
	"net/http"
	"sync"
)

type Request struct{ Method, Path, Body string }
type Response struct {
	Status      int
	Body        string
	ContentType string
}
type Handler func(Request) Response
type Router struct{ mux *http.ServeMux }

func NewRouter() *Router { return &Router{mux: http.NewServeMux()} }
func (r *Router) Handle(method, path string, handler Handler) {
	r.mux.HandleFunc(method+" "+path, func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, req.Body, 1<<20))
		if err != nil {
			http.Error(w, "Request body too large or unreadable", http.StatusRequestEntityTooLarge)
			return
		}
		res := handler(Request{Method: req.Method, Path: req.URL.Path, Body: string(body)})
		if res.Status < 100 || res.Status > 599 {
			http.Error(w, "Invalid response status", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", res.ContentType)
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
		h(conn, Request{Method: req.Method, Path: req.URL.Path})
	}))
}
func ServeHybrid(dir, port string) error {
	s, err := NewServer(ServerOptions{Dir: dir, Port: port})
	if err != nil {
		return err
	}
	return http.ListenAndServe(":"+port, defaultRouter.Handler(s))
}
