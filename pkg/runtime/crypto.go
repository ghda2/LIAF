package runtime

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	stdruntime "runtime"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// Primitivas de criptografia da stdlib. Sao o minimo para autenticacao
// (senha, token de sessao), assinatura de webhook e JWT, que bibliotecas em
// LIAF montam por cima. Nada aqui aceita parametro de seguranca vindo do
// programa: tamanho de token, custo do hash e algoritmo sao fixos, para que
// o caminho facil seja tambem o seguro.

// encodeDigest aplica a codificacao pedida. O checker ja garantiu que
// encoding e um dos literais aceitos.
func encodeDigest(sum []byte, encoding string) string {
	if encoding == "base64url" {
		return base64.RawURLEncoding.EncodeToString(sum)
	}
	return hex.EncodeToString(sum)
}

func SHA256(text, encoding string) string {
	sum := sha256.Sum256([]byte(text))
	return encodeDigest(sum[:], encoding)
}

// HMACSHA256 e o que assina webhooks (em hex) e tokens JWT HS256 (em
// base64url).
func HMACSHA256(key, text, encoding string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(text))
	return encodeDigest(mac.Sum(nil), encoding)
}

// SecureEq compara em tempo constante. Comparar uma assinatura recebida com
// eq deixaria o tempo de resposta revelar quantos caracteres ja batem.
func SecureEq(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// Base64URL sem padding, a forma usada em JWT e em URLs.
func Base64URLEncode(text string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(text))
}

// Base64URLDecode aceita a entrada com ou sem padding. O resultado precisa
// ser UTF-8: str na LIAF e texto, e a linguagem ainda nao tem tipo de bytes.
func Base64URLDecode(text string) Result[string, string] {
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(text, "="))
	if err != nil {
		return Err[string, string]("base64url-decode: invalid input")
	}
	if !utf8.Valid(data) {
		return Err[string, string]("base64url-decode: decoded data is not UTF-8 text")
	}
	return Ok[string, string](string(data))
}

// RandomToken devolve 32 bytes do gerador criptografico (256 bits) em
// base64url: 43 caracteres, seguros em URL, cookie e header.
func RandomToken() string {
	b := make([]byte, 32)
	rand.Read(b) // crypto/rand.Read nunca falha; o processo aborta se nao houver entropia
	return base64.RawURLEncoding.EncodeToString(b)
}

// --- Senhas -----------------------------------------------------------------

// Parametros do argon2id recomendados pela OWASP (2023) para quem nao pode
// dedicar 64 MiB por login: 19 MiB, 2 passadas, 1 via de paralelismo.
const (
	argonMemoryKiB = 19 * 1024
	argonTime      = 2
	argonThreads   = 1
	argonKeyLen    = 32
	argonSaltLen   = 16
)

// hashSlots limita os hashes simultaneos ao numero de CPUs. Cada um reserva
// 19 MiB; sem o limite, uma rajada de logins multiplicaria isso pelo numero
// de requisicoes e derrubaria o processo por memoria.
var hashSlots = make(chan struct{}, max(stdruntime.NumCPU(), 1))

func argonKey(password string, salt []byte, time, memory uint32, threads uint8, keyLen uint32) []byte {
	hashSlots <- struct{}{}
	defer func() { <-hashSlots }()
	return argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)
}

// PasswordHash devolve a senha no formato PHC, que carrega algoritmo,
// parametros e sal junto do hash:
//
//	$argon2id$v=19$m=19456,t=2,p=1$<sal>$<hash>
//
// Guardar os parametros no proprio texto permite endurece-los no futuro sem
// invalidar as senhas ja gravadas.
func PasswordHash(password string) string {
	salt := make([]byte, argonSaltLen)
	rand.Read(salt)
	key := argonKey(password, salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemoryKiB, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))
}

// PasswordVerify devolve false para senha errada e tambem para hash
// malformado: para quem faz login, os dois casos sao "nao entra".
func PasswordVerify(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false
	}
	// Um hash adulterado no banco nao pode pedir 64 GiB de memoria.
	if memory == 0 || memory > 1024*1024 || time == 0 || time > 16 || threads == 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) == 0 || len(want) > 128 {
		return false
	}
	got := argonKey(password, salt, time, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}
