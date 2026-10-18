package checker_test

import (
	"testing"

	"liaf/pkg/checker"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
)

func TestCheckV03TryAndRoute(t *testing.T) {
	input := `(module test-v03
  (route GET "/items/{id}"
    (params (id int))
    (returns (result str str))
    (effects fs)
    (on-err msg (return (err msg)))
    (body
      (let content str (try (fs-read-file "item.txt")))
      (return (ok content))))
  (fn main
    (params)
    (returns void)
    (effects)
    (body)))`

	l := lexer.New(input)
	p := parser.New(l, "test_v03.liaf")
	mod := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		t.Fatalf("parser diagnostics: %+v", p.Diagnostics)
	}

	info, errors := checker.Check(mod, "test_v03.liaf")
	if len(errors) > 0 {
		t.Fatalf("checker unexpected errors: %+v", errors)
	}
	if len(info.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(info.Routes))
	}
}

func TestCheckV03TryTypeMismatch(t *testing.T) {
	// try on non-result must fail
	input := `(module test-bad-try
  (fn test (params) (returns void) (effects)
    (body
      (let x int (try 42))))
  (fn main (params) (returns void) (effects) (body)))`

	l := lexer.New(input)
	p := parser.New(l, "bad_try.liaf")
	mod := p.ParseModule()

	_, errors := checker.Check(mod, "bad_try.liaf")
	if len(errors) == 0 {
		t.Fatalf("expected error for try on non-result, got none")
	}
}

// diagCodes roda o checker e devolve so os codigos, para assercoes legiveis.
func diagCodes(t *testing.T, input string) []string {
	t.Helper()
	p := parser.New(lexer.New(input), "v03.liaf")
	mod := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		t.Fatalf("parser diagnostics: %+v", p.Diagnostics)
	}
	_, errs := checker.Check(mod, "v03.liaf")
	codes := make([]string, 0, len(errs))
	for _, e := range errs {
		codes = append(codes, e.Code)
	}
	return codes
}

func hasCode(codes []string, want string) bool {
	for _, c := range codes {
		if c == want {
			return true
		}
	}
	return false
}

func TestCheckV03RoutePathParams(t *testing.T) {
	for _, tc := range []struct {
		name  string
		route string
		code  string
	}{
		{
			// {id} sem entrada em (params ...): o codegen leria o parametro do
			// corpo JSON em vez do path, entao isso tem que parar no checker.
			name:  "path param sem declaracao",
			route: `(route GET "/items/{id}" (params (other int)) (returns Response) (effects) (body (return (json-response 200 "{}"))))`,
			code:  "E_ROUTE_PARAM",
		},
		{
			name:  "path param com tipo composto",
			route: `(route GET "/items/{id}" (params (id (list int))) (returns Response) (effects) (body (return (json-response 200 "{}"))))`,
			code:  "E_ROUTE_PARAM",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codes := diagCodes(t, "(module r\n"+tc.route+"\n(fn main (params) (returns void) (effects) (body)))")
			if !hasCode(codes, tc.code) {
				t.Fatalf("esperado %s, recebido %v", tc.code, codes)
			}
		})
	}

	t.Run("path param valido passa", func(t *testing.T) {
		codes := diagCodes(t, `(module r
(route GET "/items/{id}" (params (id int)) (returns Response) (effects) (body (return (json-response 200 "{}"))))
(route GET "/users/{slug}" (params (slug str)) (returns Response) (effects) (body (return (json-response 200 "{}"))))
(fn main (params) (returns void) (effects) (body)))`)
		if len(codes) > 0 {
			t.Fatalf("erros inesperados: %v", codes)
		}
	})
}

func TestCheckV03UnusedEffects(t *testing.T) {
	// (effects fs) numa funcao que nunca toca o disco engana quem le a
	// assinatura, inclusive um modelo gerando codigo a partir dela.
	codes := diagCodes(t, `(module dead
(fn greet (params (who str)) (returns str) (effects fs)
  (body (return (concat "oi " who))))
(fn main (params) (returns void) (effects io)
  (body (do (println (greet "mundo"))))))`)
	if !hasCode(codes, "E_UNUSED_EFFECT") {
		t.Fatalf("esperado E_UNUSED_EFFECT, recebido %v", codes)
	}

	t.Run("efeito transitivo conta como uso", func(t *testing.T) {
		codes := diagCodes(t, `(module transitivo
(fn ler (params) (returns (result str str)) (effects fs)
  (body (return (fs-read-file "x.txt"))))
(fn main (params) (returns void) (effects fs io)
  (body
    (match (ler)
      (ok texto (do (println texto)))
      (err msg (do (println msg)))))))`)
		if len(codes) > 0 {
			t.Fatalf("chamar uma funcao com efeito fs deve contar como uso: %v", codes)
		}
	})

	t.Run("efeito de rota nao usado", func(t *testing.T) {
		codes := diagCodes(t, `(module rota
(route GET "/ping" (params) (returns Response) (effects net)
  (body (return (json-response 200 "{}"))))
(fn main (params) (returns void) (effects) (body)))`)
		if !hasCode(codes, "E_UNUSED_EFFECT") {
			t.Fatalf("esperado E_UNUSED_EFFECT na rota, recebido %v", codes)
		}
	})
}

// TestCheckV03JSONIsPure cobre o RFC secao 3.3: json-encode e json-decode nao
// fazem I/O, operam sobre strings em memoria. Exigir `io` delas fazia toda
// funcao que so serializa parecer efetuosa.
func TestCheckV03JSONIsPure(t *testing.T) {
	codes := diagCodes(t, `(module puro
(struct Task (fields (id int) (title str)))
(fn serializa (params (t Task)) (returns (result str str)) (effects)
  (body (return (json-encode t))))
(fn desserializa (params (texto str)) (returns (result Task str)) (effects)
  (body (return (json-decode texto Task))))
(fn main (params) (returns void) (effects) (body)))`)
	if len(codes) > 0 {
		t.Fatalf("json-encode/json-decode devem ser puros: %v", codes)
	}
}

// TestCheckV03JSONDecodeList cobre o RFC secao 3.2: json-decode aceita tipos
// compostos, nao so nomes de tipo.
func TestCheckV03JSONDecodeList(t *testing.T) {
	codes := diagCodes(t, `(module lista
(struct Task (fields (id int)))
(fn ler (params (texto str)) (returns (result (list Task) str)) (effects)
  (body (return (json-decode texto (list Task)))))
(fn main (params) (returns void) (effects) (body)))`)
	if len(codes) > 0 {
		t.Fatalf("json-decode deve aceitar (list T): %v", codes)
	}
}

// TestCheckV03WriteOpsReturnVoid cobre o RFC secao 3.1: as escritas nao
// carregam valor no sucesso, entao nao existe o caminho `ok false`.
func TestCheckV03WriteOpsReturnVoid(t *testing.T) {
	for _, op := range []string{
		`(fs-write-file "a.txt" "x")`,
		`(fs-rename "a.txt" "b.txt")`,
		`(fs-write-atomic "a.txt" "x")`,
		`(fs-remove "a.txt")`,
	} {
		t.Run(op, func(t *testing.T) {
			codes := diagCodes(t, `(module escrita
(fn usa (params) (returns void) (effects fs io)
  (body
    (match `+op+`
      (ok feito (do (println feito)))
      (err m (do (println m))))))
(fn main (params) (returns void) (effects) (body)))`)
			if !hasCode(codes, "E_TYPE_MISMATCH") {
				t.Fatalf("usar o valor de ok deve falhar para (result void str): %v", codes)
			}
		})
	}
}
