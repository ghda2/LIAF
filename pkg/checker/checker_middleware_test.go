package checker_test

import "testing"

func TestMiddlewareChecker(t *testing.T) {
	const guard = `
middleware guard(req Request) (result Request Response)
  ok(req)
end
`
	const main = `
fn main() void
end
`
	cases := []struct {
		name string
		src  string
		want string // "" = sem erros
	}{
		{"valido", guard + `
route GET "/p" with(guard) () Response
  json-response(200, "{}")
end` + main, ""},
		{"nao_declarado", `
route GET "/p" with(sumiu) () Response
  json-response(200, "{}")
end` + main, "E_UNKNOWN_MIDDLEWARE"},
		{"fn_comum_no_with", `
fn guard(req Request) (result Request Response)
  ok(req)
end
route GET "/p" with(guard) () Response
  json-response(200, "{}")
end` + main, "E_UNKNOWN_MIDDLEWARE"},
		{"repetido", guard + `
route GET "/p" with(guard, guard) () Response
  json-response(200, "{}")
end` + main, "E_DUPLICATE_MIDDLEWARE"},
		{"retorno_errado", `
middleware guard(req Request) (result Request str)
  ok(req)
end` + main, "E_MIDDLEWARE_SIGNATURE"},
		{"parametro_errado", `
middleware guard(token str) (result Request Response)
  err(json-error(401, token))
end` + main, "E_MIDDLEWARE_SIGNATURE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			codes := parseAndCheck(t, "module m\n"+tc.src, "m.liaf")
			if tc.want == "" {
				if len(codes) != 0 {
					t.Fatalf("esperado sem erros, obteve %v", codes)
				}
				return
			}
			if !contains(codes, tc.want) {
				t.Fatalf("esperado %s, obteve %v", tc.want, codes)
			}
		})
	}
}
