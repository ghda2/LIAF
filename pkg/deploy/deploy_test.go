package deploy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnitValidation(t *testing.T) {
	dir := t.TempDir()
	s, err := Unit("news", filepath.Join(dir, "app with space.exe"), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "EnvironmentFile=-/etc/liaf/news.env") || !strings.Contains(s, `ExecStart="`) {
		t.Fatal(s)
	}
	for _, n := range []string{"../escape", "a\nExecStart=bad", "", "a.service"} {
		if _, err = Unit(n, filepath.Join(dir, "app"), dir); err == nil {
			t.Fatal(n)
		}
	}
}
func TestCaddyBinding(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/config/apps/http/servers/srv0/routes" {
			t.Error(r.URL.Path)
		}
		if r.Method == "GET" {
			w.Header().Set("ETag", `"version1"`)
			w.Write([]byte(`[{"@id":"other"},{"@id":"liaf-example.test"}]`))
			return
		}
		if r.Method != "PATCH" || r.Header.Get("If-Match") != `"version1"` {
			t.Error(r.Method, r.Header)
		}
		var routes []map[string]any
		if err := json.NewDecoder(r.Body).Decode(&routes); err != nil {
			t.Fatal(err)
		}
		if len(routes) != 2 || routes[0]["@id"] != "liaf-example.test" || routes[1]["@id"] != "other" {
			t.Error(routes)
		}
	}))
	defer srv.Close()
	if err := BindCaddy(context.Background(), srv.Client(), srv.URL, "srv0", "example.test", "127.0.0.1:7070"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal(calls)
	}
}
