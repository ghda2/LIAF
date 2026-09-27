package runtime

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/crypto/argon2"
)

// Vetores publicados: FIPS 180-2 (SHA-256 de "abc") e RFC 4231, caso 2.
func TestDigestKnownVectors(t *testing.T) {
	if got := SHA256("abc", "hex"); got != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Errorf("sha256 hex: %s", got)
	}
	if got := SHA256("abc", "base64url"); got != "ungWv48Bz-pBQUDeXa4iI7ADYaOWF3qctBD_YfIAFa0" {
		t.Errorf("sha256 base64url: %s", got)
	}
	if got := HMACSHA256("Jefe", "what do ya want for nothing?", "hex"); got != "5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843" {
		t.Errorf("hmac hex: %s", got)
	}
}

// Assinatura de um JWT HS256 conhecido (exemplo de jwt.io, segredo
// "your-256-bit-secret"): e o que um jwt.liaf vai montar com estes builtins.
func TestHMACSignsJWT(t *testing.T) {
	header := Base64URLEncode(`{"alg":"HS256","typ":"JWT"}`)
	payload := Base64URLEncode(`{"sub":"1234567890","name":"John Doe","iat":1516239022}`)
	sig := HMACSHA256("your-256-bit-secret", header+"."+payload, "base64url")
	if sig != "SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c" {
		t.Errorf("assinatura JWT: %s", sig)
	}
}

func TestBase64URL(t *testing.T) {
	for _, text := range []string{"", "a", "ab", "abc", "ação ✓", "?>>"} {
		enc := Base64URLEncode(text)
		if strings.ContainsAny(enc, "+/=") {
			t.Errorf("%q: saida fora do alfabeto base64url: %s", text, enc)
		}
		if r := Base64URLDecode(enc); !r.OK || r.Value != text {
			t.Errorf("%q: ida e volta: %+v", text, r)
		}
	}
	// Com padding tambem decodifica: ha emissores que o mantem.
	if r := Base64URLDecode("YQ=="); !r.OK || r.Value != "a" {
		t.Errorf("com padding: %+v", r)
	}
	for _, bad := range []string{"não é base64", "YQ=x", "/w"} {
		if r := Base64URLDecode(bad); r.OK {
			t.Errorf("%q: aceito como valido: %+v", bad, r)
		}
	}
	// 0xff nao e UTF-8: str da LIAF e texto.
	if r := Base64URLDecode("_w"); r.OK {
		t.Errorf("bytes que nao sao UTF-8 foram aceitos: %+v", r)
	}
}

func TestSecureEq(t *testing.T) {
	if !SecureEq("abc", "abc") || SecureEq("abc", "abd") || SecureEq("abc", "ab") || SecureEq("", "a") {
		t.Error("secure-eq devolveu o resultado errado")
	}
}

func TestRandomToken(t *testing.T) {
	format := regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		tok := RandomToken()
		if !format.MatchString(tok) {
			t.Fatalf("formato: %q", tok)
		}
		if seen[tok] {
			t.Fatalf("token repetido: %s", tok)
		}
		seen[tok] = true
	}
}

func TestPasswordHashAndVerify(t *testing.T) {
	h1 := PasswordHash("correct horse")
	h2 := PasswordHash("correct horse")
	if !strings.HasPrefix(h1, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("formato PHC: %s", h1)
	}
	if h1 == h2 {
		t.Error("mesmo hash para a mesma senha: o sal nao esta sendo sorteado")
	}
	if !PasswordVerify("correct horse", h1) || !PasswordVerify("correct horse", h2) {
		t.Error("senha certa recusada")
	}
	if PasswordVerify("correct horsE", h1) || PasswordVerify("", h1) {
		t.Error("senha errada aceita")
	}
}

func TestPasswordVerifyRejectsMalformedHash(t *testing.T) {
	good := PasswordHash("s3nha")
	parts := strings.Split(good, "$")
	for name, bad := range map[string]string{
		"vazio":            "",
		"texto puro":       "s3nha",
		"outro algoritmo":  strings.Replace(good, "argon2id", "argon2i", 1),
		"versao errada":    strings.Replace(good, "v=19", "v=16", 1),
		"memoria absurda":  strings.Replace(good, "m=19456", "m=99999999", 1),
		"hash adulterado":  strings.Join(append(parts[:5:5], "AAAA"+parts[5][4:]), "$"),
		"sal nao e base64": strings.Join([]string{"", parts[1], parts[2], parts[3], "!!", parts[5]}, "$"),
		"campos faltando":  strings.Join(parts[:5], "$"),
	} {
		if PasswordVerify("s3nha", bad) {
			t.Errorf("%s: aceito", name)
		}
	}
	// Parametros mais fracos que os atuais continuam verificando: o custo
	// vem do proprio hash gravado, entao endurecer o padrao nao invalida
	// senhas antigas.
	salt := []byte("saltsalt")
	key := argon2.IDKey([]byte("s3nha"), salt, 1, 8, 1, 32)
	weak := "$argon2id$v=19$m=8,t=1,p=1$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key)
	if !PasswordVerify("s3nha", weak) {
		t.Error("hash com parametros antigos deixou de verificar")
	}
}
