package loader_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"liaf/pkg/checker"
	"liaf/pkg/loader"
	"liaf/std"
)

// write grava os arquivos num diretorio temporario e devolve o caminho do
// primeiro nome passado.
func write(t *testing.T, files map[string]string, first string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, first)
}

func diagWith(t *testing.T, path, code string, l *loader.Loader) string {
	t.Helper()
	if l == nil {
		l = loader.New()
	}
	_, diags := l.Load(path)
	for _, d := range diags {
		if d.Code == code {
			return d.File + ":" + d.Message
		}
	}
	t.Fatalf("esperado %s, recebido %+v", code, diags)
	return ""
}

func loadOK(t *testing.T, path string, l *loader.Loader) {
	t.Helper()
	if l == nil {
		l = loader.New()
	}
	mod, diags := l.Load(path)
	if len(diags) > 0 {
		t.Fatalf("loader: %+v", diags)
	}
	if _, diags := checker.Check(mod, path); len(diags) > 0 {
		t.Fatalf("checker: %+v", diags)
	}
}

// --- Biblioteca padrao --------------------------------------------------------

// A std de verdade, embutida no pacote: std/auth tem de carregar e passar no
// checker junto com o programa, de qualquer diretorio.
func TestStdEmbeddedAuth(t *testing.T) {
	loadOK(t, write(t, map[string]string{"main.liaf": `(module main
  (import "std/auth")
  (route GET "/eu" (params (req Request)) (returns Response) (effects)
    (on-err m (return (json-response 401 "{}")))
    (body
      (let token str (try (auth-bearer-token req)))
      (return (json-response 200 token))))
  (fn main (params) (returns void) (effects) (body)))`}, "main.liaf"), nil)
}

func TestStdUnknownModuleListsAvailable(t *testing.T) {
	msg := diagWith(t, write(t, map[string]string{"main.liaf": `(module main
  (import "std/jwtt")
  (fn main (params) (returns void) (effects) (body)))`}, "main.liaf"), "E_IMPORT_NOT_FOUND", nil)
	// A lista de modulos e o que deixa um modelo se corrigir sozinho.
	if !strings.Contains(msg, "std/auth") {
		t.Errorf("mensagem sem a lista de modulos: %s", msg)
	}
	if !strings.Contains(msg, "main.liaf") {
		t.Errorf("erro sem o arquivo que importou: %s", msg)
	}
}

func TestStdRejectsPathTricks(t *testing.T) {
	for _, imp := range []string{"std/", "std/../main", "std/sub/x", `std/..\x`} {
		diagWith(t, write(t, map[string]string{"main.liaf": `(module main
  (import "` + strings.ReplaceAll(imp, `\`, `\\`) + `")
  (fn main (params) (returns void) (effects) (body)))`}, "main.liaf"), "E_IMPORT_NOT_FOUND", nil)
	}
}

func fakeStd(files map[string]string) *loader.Loader {
	fsys := fstest.MapFS{}
	for name, content := range files {
		fsys[name] = &fstest.MapFile{Data: []byte(content)}
	}
	l := loader.New()
	l.Std = fsys
	return l
}

func TestStdModuleImportsStd(t *testing.T) {
	l := fakeStd(map[string]string{
		"base.liaf": `(module base (fn base-um (params) (returns int) (effects) (body (return 1))))`,
		"soma.liaf": `(module soma (import "std/base") (fn soma-dois (params) (returns int) (effects) (body (return (add (base-um) (base-um))))))`,
	})
	loadOK(t, write(t, map[string]string{"main.liaf": `(module main
  (import "std/soma")
  (fn main (params) (returns void) (effects io) (body (println (soma-dois)))))`}, "main.liaf"), l)
}

// Um modulo da std nao pode depender do disco de quem o importa.
func TestStdModuleCannotImportRelative(t *testing.T) {
	l := fakeStd(map[string]string{
		"mau.liaf": `(module mau (import "./segredo") (fn mau-x (params) (returns int) (effects) (body (return 1))))`,
	})
	msg := diagWith(t, write(t, map[string]string{"main.liaf": `(module main
  (import "std/mau")
  (fn main (params) (returns void) (effects) (body)))`}, "main.liaf"), "E_STD_RELATIVE_IMPORT", l)
	if !strings.HasPrefix(msg, "std/mau.liaf:") {
		t.Errorf("erro devia apontar o modulo da std: %s", msg)
	}
}

// Dois arquivos importando a mesma std (diamante) nao e colisao.
func TestStdDiamondIsNotACollision(t *testing.T) {
	loadOK(t, write(t, map[string]string{
		"main.liaf": `(module main
  (import "std/auth")
  (import "./painel.liaf")
  (fn main (params) (returns void) (effects) (body)))`,
		"painel.liaf": `(module painel
  (import "std/auth")
  (fn painel-token (params (req Request)) (returns (result str str)) (effects)
    (body (return (auth-bearer-token req)))))`,
	}, "main.liaf"), nil)
}

// Uma pasta local chamada std continua acessivel, com "./".
func TestLocalStdFolderNeedsDotSlash(t *testing.T) {
	loadOK(t, write(t, map[string]string{
		"main.liaf":     `(module main (import "./std/util") (fn main (params) (returns void) (effects io) (body (println (util-x)))))`,
		"std/util.liaf": `(module util (fn util-x (params) (returns int) (effects) (body (return 1))))`,
	}, "main.liaf"), nil)
}

// --- Colisoes -----------------------------------------------------------------

func TestCollisionWithStdPointsAtUserFile(t *testing.T) {
	msg := diagWith(t, write(t, map[string]string{"main.liaf": `(module main
  (import "std/auth")
  (fn auth-bearer-token (params (req Request)) (returns (result str str)) (effects)
    (body (return (ok "x"))))
  (fn main (params) (returns void) (effects) (body)))`}, "main.liaf"), "E_DUPLICATE_DECL", nil)
	for _, want := range []string{"main.liaf:", "std/auth.liaf:", "biblioteca padrão"} {
		if !strings.Contains(msg, want) {
			t.Errorf("mensagem sem %q: %s", want, msg)
		}
	}
}

func TestCollisionBetweenUserFiles(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"fn com fn": {
			"main.liaf": `(module main (import "./a.liaf") (fn dobrar (params (n int)) (returns int) (effects) (body (return n))) (fn main (params) (returns void) (effects) (body)))`,
			"a.liaf":    `(module a (fn dobrar (params (n int)) (returns int) (effects) (body (return (add n n)))))`,
		},
		// fn e struct dividem o espaco de nomes, como no checker.
		"struct com fn": {
			"main.liaf": `(module main (import "./a.liaf") (fn User (params) (returns int) (effects) (body (return 1))) (fn main (params) (returns void) (effects) (body)))`,
			"a.liaf":    `(module a (struct User (fields (name str))))`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			msg := diagWith(t, write(t, files, "main.liaf"), "E_DUPLICATE_DECL", nil)
			if !strings.Contains(msg, "a.liaf:1:") {
				t.Errorf("mensagem sem a outra declaracao: %s", msg)
			}
		})
	}
}

// Nome repetido no mesmo arquivo continua sendo do checker.
func TestSameFileSymbolIsLeftToChecker(t *testing.T) {
	path := write(t, map[string]string{"main.liaf": `(module main
  (fn f (params) (returns int) (effects) (body (return 1)))
  (fn f (params) (returns int) (effects) (body (return 2)))
  (fn main (params) (returns void) (effects) (body)))`}, "main.liaf")
	mod, diags := loader.Load(path)
	if len(diags) > 0 {
		t.Fatalf("loader nao devia reclamar: %+v", diags)
	}
	_, checkDiags := checker.Check(mod, path)
	found := false
	for _, d := range checkDiags {
		found = found || d.Code == "E_DUPLICATE_SYMBOL"
	}
	if !found {
		t.Fatalf("checker nao reportou: %+v", checkDiags)
	}
}

func TestDuplicateRoutes(t *testing.T) {
	route := func(method, path string) string {
		return `(route ` + method + ` "` + path + `" (params) (returns void) (effects) (body))`
	}
	ws := `(ws-route "/x" (params (conn WSConn)) (effects net) (on-message t (do (ws-join conn t))))`
	for name, tc := range map[string]struct {
		main, other string
		dup         bool
	}{
		"mesmo arquivo":        {route("GET", "/x") + route("GET", "/x"), "", true},
		"arquivos diferentes":  {route("POST", "/x"), route("POST", "/x"), true},
		"metodo em minusculas": {route("GET", "/x"), route("get", "/x"), true},
		"ws-route ocupa o GET": {route("GET", "/x") + ws, "", true},
		"metodos diferentes":   {route("GET", "/x") + route("POST", "/x"), "", false},
		"caminhos diferentes":  {route("GET", "/x"), route("GET", "/y"), false},
	} {
		t.Run(name, func(t *testing.T) {
			path := write(t, map[string]string{
				"main.liaf":  `(module main (import "./outro.liaf") ` + tc.main + ` (fn main (params) (returns void) (effects) (body)))`,
				"outro.liaf": `(module outro ` + tc.other + `)`,
			}, "main.liaf")
			_, diags := loader.Load(path)
			got := false
			for _, d := range diags {
				got = got || d.Code == "E_DUPLICATE_ROUTE"
			}
			if got != tc.dup {
				t.Fatalf("E_DUPLICATE_ROUTE=%v, esperado %v: %+v", got, tc.dup, diags)
			}
		})
	}
}

// Cada modulo da std, sozinho, carrega e passa no checker. Um modulo que
// nenhum exemplo usa ainda assim quebraria o programa de quem o importar.
func TestStdEveryModuleChecks(t *testing.T) {
	names, err := fs.Glob(std.FS, "*.liaf")
	if err != nil || len(names) == 0 {
		t.Fatalf("std vazia: %v", err)
	}
	for _, name := range names {
		mod := strings.TrimSuffix(name, ".liaf")
		t.Run(mod, func(t *testing.T) {
			loadOK(t, write(t, map[string]string{"main.liaf": `(module main
  (import "std/` + mod + `")
  (fn main (params) (returns void) (effects) (body)))`}, "main.liaf"), nil)
		})
	}
}
