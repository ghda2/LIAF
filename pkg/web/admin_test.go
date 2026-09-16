package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestAdminAuthenticationAndLimits(t *testing.T) {
	t.Setenv("LIAF_DEPLOY_TOKEN", "")
	s, err := NewServer(ServerOptions{Dir: t.TempDir(), MaxPublishSize: 8})
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, route, token, path, body string) int {
		r := httptest.NewRequest(method, route, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("X-LIAF-Path", path)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w.Code
	}
	if got := request("POST", "/_liaf/publish", "", "x.html", "x"); got != 503 {
		t.Fatal(got)
	}
	s.options.DeployToken = "secret"
	for _, route := range []string{"/_liaf/publish", "/_liaf/reload"} {
		if got := request("GET", route, "secret", "x.html", ""); got != 405 {
			t.Fatal(got)
		}
		if got := request("POST", route, "wrong", "x.html", ""); got != 401 {
			t.Fatal(got)
		}
	}
	for _, p := range []string{"../outside", "a/../../outside", `a\..\outside`, "C:/outside", ".env", "a/.secret"} {
		if got := request("POST", "/_liaf/publish", "secret", p, "x"); got != 400 {
			t.Fatalf("%q: %d", p, got)
		}
	}
	if got := request("POST", "/_liaf/publish", "secret", "large.txt", "123456789"); got != 413 {
		t.Fatal(got)
	}
	if got := request("POST", "/_liaf/reload", "secret", "", ""); got != 200 {
		t.Fatal(got)
	}
}

func TestConcurrentPublishAndThreshold(t *testing.T) {
	dir := t.TempDir()
	s, err := NewServer(ServerOptions{Dir: dir, DeployToken: "secret", MaxRAMAssetSize: 8})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := httptest.NewRequest("POST", "/_liaf/publish", strings.NewReader("1234567890"))
			r.Header.Set("Authorization", "Bearer secret")
			r.Header.Set("X-LIAF-Path", fmt.Sprintf("file-%d.bin", i))
			w := httptest.NewRecorder()
			s.ServeHTTP(w, r)
			if w.Code != 200 {
				t.Errorf("publish %d: %s", w.Code, w.Body)
			}
		}(i)
	}
	wg.Wait()
	for i := 0; i < 12; i++ {
		a, ok := s.CurrentCache().Get(fmt.Sprintf("/file-%d.bin", i))
		if !ok || !a.IsDisk || len(a.Content) != 0 {
			t.Fatalf("missing or RAM asset %d", i)
		}
	}
	r := httptest.NewRequest("GET", "/file-0.bin", nil)
	r.Header.Set("Range", "bytes=2-4")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 206 || w.Body.String() != "345" {
		t.Fatal(w.Code, w.Body.String())
	}
	r = httptest.NewRequest("GET", "/file-0.bin", nil)
	r.Header.Set("Range", "bytes=100-200")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 416 {
		t.Fatal(w.Code)
	}
}

func TestTemplatesCannotEscapeRoot(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "public")
	os.Mkdir(dir, 0755)
	os.WriteFile(filepath.Join(parent, "secret.txt"), []byte("SECRET_OUTSIDE_ROOT"), 0600)
	os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<!-- include "../secret.txt" -->`), 0600)
	s, err := NewServer(ServerOptions{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if strings.Contains(w.Body.String(), "SECRET_OUTSIDE_ROOT") {
		t.Fatal("template escaped root")
	}
}

func TestHybridRouter(t *testing.T) {
	router := NewRouter()
	router.Handle("GET", "/api/users", func(r Request) Response { return JSONResponse(200, `{"path":"`+r.Path+`"}`) })
	h := router.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "static") }))
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{{"GET", "/api/users", `{"path":"/api/users"}`, 200}, {"POST", "/api/users", "", 405}, {"GET", "/page", "static", 200}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.status || (tc.body != "" && w.Body.String() != tc.body) {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
