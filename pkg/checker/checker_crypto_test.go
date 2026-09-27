package checker_test

import "testing"

func cryptoModule(effects, body string) string {
	return `(module c
(fn f (params (s str)) (returns bool) (effects ` + effects + `)
  (body
    ` + body + `
    (return true)))
(fn main (params) (returns void) (effects) (body)))`
}

func TestCheckCryptoAcceptsValidUsage(t *testing.T) {
	codes := diagCodes(t, cryptoModule("rand", `
    (let a str (sha256 s "hex"))
    (let b str (hmac-sha256 "chave" s "base64url"))
    (let c bool (secure-eq a b))
    (let d str (base64url-encode s))
    (let e (result str str) (base64url-decode d))
    (match e (ok x (let y str x)) (err m (let z str m)))
    (let tok str (random-token))
    (let h str (password-hash s))
    (let ok bool (password-verify s h))`))
	if len(codes) > 0 {
		t.Fatalf("uso valido foi recusado: %v", codes)
	}
}

func TestCheckCryptoRejectsMisuse(t *testing.T) {
	for _, tc := range []struct{ name, effects, body, want string }{
		// "base64" padrao produziria +, / e =, que quebram um JWT.
		{"codificacao desconhecida", "", `(let a str (sha256 s "base64"))`, "E_UNKNOWN_ENCODING"},
		{"codificacao em maiusculas", "", `(let a str (hmac-sha256 "k" s "HEX"))`, "E_UNKNOWN_ENCODING"},
		{"codificacao nao literal", "", `(let a str (sha256 s s))`, "E_UNKNOWN_ENCODING"},
		{"chave nao e str", "", `(let a str (hmac-sha256 1 s "hex"))`, "E_TYPE_MISMATCH"},
		{"faltou a codificacao", "", `(let a str (sha256 s))`, "E_WRONG_ARITY"},
		{"decode sem tratar o result", "", `(let a str (base64url-decode s))`, "E_TYPE_MISMATCH"},
		{"token sem o efeito rand", "", `(let a str (random-token))`, "E_UNDECLARED_EFFECT"},
		{"hash de senha sem o efeito rand", "", `(let a str (password-hash s))`, "E_UNDECLARED_EFFECT"},
		{"verify devolve bool", "", `(let a str (password-verify s s))`, "E_TYPE_MISMATCH"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codes := diagCodes(t, cryptoModule(tc.effects, tc.body))
			if !hasCode(codes, tc.want) {
				t.Fatalf("esperado %s, recebido %v", tc.want, codes)
			}
		})
	}
}
