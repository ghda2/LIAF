package packages

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func tarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte(body))
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func TestScan(t *testing.T) {
	got := Scan(`#import "@preview/tiaoma:0.3.0": qrcode`, `// @preview/cetz:0.4.2 e de novo @preview/tiaoma:0.3.0`, "email@exemplo.com")
	if len(got) != 2 || got[0].String() != "@preview/cetz:0.4.2" || got[1].String() != "@preview/tiaoma:0.3.0" {
		t.Fatalf("Scan = %v", got)
	}
}

// Servidor falso do Typst Universe: "a" depende de "b".
func universe(t *testing.T, hits *int32) *httptest.Server {
	archives := map[string][]byte{
		"/preview/a-1.0.0.tar.gz": tarGz(t, map[string]string{
			"typst.toml": "[package]\nname = \"a\"",
			"lib.typ":    `#import "@preview/b:2.0.0": f` + "\n#let g() = f()",
		}),
		"/preview/b-2.0.0.tar.gz": tarGz(t, map[string]string{
			"typst.toml": "[package]\nname = \"b\"",
			"lib.typ":    "#let f() = [b]",
		}),
		"/preview/mau-1.0.0.tar.gz": tarGz(t, map[string]string{
			"typst.toml":   "",
			"../escape.typ": "x",
		}),
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		body, ok := archives[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(body)
	}))
}

func TestResolveTransitiveCachedAndLocked(t *testing.T) {
	var hits int32
	srv := universe(t, &hits)
	defer srv.Close()
	old := Registry
	Registry = srv.URL
	defer func() { Registry = old }()

	cache := t.TempDir()
	lock := map[Spec]string{}
	r := &Resolver{Cache: cache, Lock: lock}
	all, err := r.Resolve(Scan("@preview/a:1.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[1].Name != "b" {
		t.Fatalf("dependência de a não resolvida: %v", all)
	}
	if _, err := os.Stat(filepath.Join(cache, "preview", "b", "2.0.0", "lib.typ")); err != nil {
		t.Fatal(err)
	}
	if len(lock) != 2 {
		t.Fatalf("trava deveria ter 2 entradas: %v", lock)
	}

	// Segunda vez: vem do cache, sem rede.
	before := atomic.LoadInt32(&hits)
	if _, err := r.Resolve(Scan("@preview/a:1.0.0")); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&hits) != before {
		t.Fatal("pacote em cache foi baixado de novo")
	}

	// Trava divergente: o pacote mudou no servidor.
	bad := map[Spec]string{{Namespace: "preview", Name: "b", Version: "2.0.0"}: strings.Repeat("0", 64)}
	r2 := &Resolver{Cache: t.TempDir(), Lock: bad}
	if _, err := r2.Resolve(Scan("@preview/b:2.0.0")); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("esperava erro de SHA-256, veio %v", err)
	}
}

func TestResolveRejectsTraversal(t *testing.T) {
	var hits int32
	srv := universe(t, &hits)
	defer srv.Close()
	old := Registry
	Registry = srv.URL
	defer func() { Registry = old }()

	cache := t.TempDir()
	r := &Resolver{Cache: cache}
	if _, err := r.Resolve(Scan("@preview/mau:1.0.0")); err == nil {
		t.Fatal("pacote com ../ foi aceito")
	}
	if _, err := os.Stat(filepath.Join(cache, "preview", "mau", "escape.typ")); err == nil {
		t.Fatal("arquivo escapou do diretório do pacote")
	}
}

func TestLockRoundTrip(t *testing.T) {
	file := filepath.Join(t.TempDir(), "liaf-typst.lock")
	lock := map[Spec]string{
		{Namespace: "preview", Name: "z", Version: "1.0.0"}: "aa",
		{Namespace: "preview", Name: "a", Version: "0.1.0"}: "bb",
	}
	if err := WriteLock(file, lock); err != nil {
		t.Fatal(err)
	}
	got, err := ReadLock(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[Spec{"preview", "a", "0.1.0"}] != "bb" {
		t.Fatalf("ReadLock = %v", got)
	}
	b, _ := os.ReadFile(file)
	if strings.Index(string(b), "@preview/a") > strings.Index(string(b), "@preview/z") {
		t.Fatal("trava fora de ordem")
	}
}
