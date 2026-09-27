package checker_test

import "testing"

// request-header, request-query e response-set-header: o compilador tem de
// recusar o uso errado antes de o servidor subir.

func httpModule(route string) string {
	return `(module api
` + route + `
(fn main (params) (returns void) (effects) (body)))`
}

func TestCheckHTTPHeadersAcceptsRouteUsage(t *testing.T) {
	codes := diagCodes(t, httpModule(`
(route GET "/api/pedidos" (params (req Request)) (returns Response) (effects)
  (on-err m (return (json-response 401 "{}")))
  (body
    (let token str (try (request-header req "Authorization")))
    (let status str (try (request-query req "status")))
    (return (response-set-header (json-response 200 (concat token status)) "Cache-Control" "no-store"))))`))
	if len(codes) > 0 {
		t.Fatalf("uso valido foi recusado: %v", codes)
	}
}

func TestCheckHTTPHeadersRejectsMisuse(t *testing.T) {
	for _, tc := range []struct{ name, expr, want string }{
		{"header usado como str sem tratar o result", `(let t str (request-header req "Authorization"))`, "E_TYPE_MISMATCH"},
		{"nome do header nao e str", `(let t (result str str) (request-header req 1))`, "E_TYPE_MISMATCH"},
		{"primeiro argumento nao e Request", `(let t (result str str) (request-query "status" "x"))`, "E_TYPE_MISMATCH"},
		{"faltou argumento", `(let t (result str str) (request-query req))`, "E_WRONG_ARITY"},
		{"set-header fora de Response", `(let r Response (response-set-header 200 "A" "b"))`, "E_TYPE_MISMATCH"},
		{"valor do header nao e str", `(let r Response (response-set-header (json-response 200 "{}") "A" 1))`, "E_TYPE_MISMATCH"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codes := diagCodes(t, httpModule(`
(route GET "/x" (params (req Request)) (returns Response) (effects)
  (body
    `+tc.expr+`
    (return (json-response 200 "{}"))))`))
			if !hasCode(codes, tc.want) {
				t.Fatalf("esperado %s, recebido %v", tc.want, codes)
			}
		})
	}
}

func fetchModule(effects, body string) string {
	return `(module f
(fn chamar (params (url str) (metodo str)) (returns (result int str)) (effects ` + effects + `)
  (body
    ` + body + `))
(fn main (params) (returns void) (effects) (body)))`
}

func TestCheckHTTPFetchAcceptsValidUsage(t *testing.T) {
	codes := diagCodes(t, fetchModule("net", `
    (let r HttpReply (try (http-fetch "POST" url (list "Authorization: Bearer x") "{}")))
    (let r2 HttpReply (try (http-fetch metodo url (list) "")))
    (let tipo str "")
    (match (reply-header r "Content-Type") (ok c (set tipo c)) (err e (set tipo "")))
    (let corpo str (concat tipo (reply-body r2)))
    (return (ok (reply-status r)))`))
	if len(codes) > 0 {
		t.Fatalf("uso valido foi recusado: %v", codes)
	}
}

func TestCheckHTTPFetchRejectsMisuse(t *testing.T) {
	for _, tc := range []struct{ name, effects, body, want string }{
		{"sem o efeito net", "", `(let r HttpReply (try (http-fetch "GET" url (list) ""))) (return (ok 1))`, "E_UNDECLARED_EFFECT"},
		{"metodo literal desconhecido", "net", `(let r HttpReply (try (http-fetch "POTS" url (list) ""))) (return (ok 1))`, "E_UNKNOWN_METHOD"},
		{"metodo em minusculas", "net", `(let r HttpReply (try (http-fetch "get" url (list) ""))) (return (ok 1))`, "E_UNKNOWN_METHOD"},
		{"headers como str", "net", `(let r HttpReply (try (http-fetch "GET" url "Authorization: x" ""))) (return (ok 1))`, "E_TYPE_MISMATCH"},
		{"resultado usado sem tratar", "net", `(let r HttpReply (http-fetch "GET" url (list) "")) (return (ok 1))`, "E_TYPE_MISMATCH"},
		{"faltou o corpo", "net", `(let r HttpReply (try (http-fetch "GET" url (list)))) (return (ok 1))`, "E_WRONG_ARITY"},
		{"status nao e str", "net", `(let r HttpReply (try (http-fetch "GET" url (list) ""))) (let s str (reply-status r)) (return (ok 1))`, "E_TYPE_MISMATCH"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codes := diagCodes(t, fetchModule(tc.effects, tc.body))
			if !hasCode(codes, tc.want) {
				t.Fatalf("esperado %s, recebido %v", tc.want, codes)
			}
		})
	}
}
