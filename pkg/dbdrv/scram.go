package dbdrv

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// SCRAM-SHA-256 (RFC 5802 / RFC 7677) e o metodo padrao do PostgreSQL desde a
// versao 14. Esta implementacao cobre o lado cliente sem canal de binding
// (gs2-header "n,,"), que e o que o servidor oferece em SCRAM-SHA-256; o
// SCRAM-SHA-256-PLUS exige tls-server-end-point e nao esta suportado.

// pbkdf2SHA256 deriva keyLen bytes. Sao poucas linhas e evita puxar
// golang.org/x/crypto/pbkdf2 so por isto.
func pbkdf2SHA256(password, salt []byte, iterations, keyLen int) []byte {
	out := make([]byte, 0, keyLen)
	block := make([]byte, 4)
	for counter := 1; len(out) < keyLen; counter++ {
		binary.BigEndian.PutUint32(block, uint32(counter))
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write(block)
		u := mac.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for i := 1; i < iterations; i++ {
			mac := hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// scramClient guarda o estado entre as tres trocas de mensagem.
type scramClient struct {
	password    string
	nonce       string
	firstBare   string
	authMessage string
	serverKey   []byte
}

func newSCRAMClient(password string) (*scramClient, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("could not generate SCRAM nonce: %w", err)
	}
	nonce := base64.StdEncoding.EncodeToString(raw)
	// O usuario vai no StartupMessage do PostgreSQL, entao n= fica vazio aqui.
	return &scramClient{password: password, nonce: nonce, firstBare: "n=,r=" + nonce}, nil
}

// first devolve a client-first-message completa, com o gs2-header.
func (s *scramClient) first() string { return "n,," + s.firstBare }

// final consome a server-first-message e devolve a client-final-message.
func (s *scramClient) final(serverFirst string) (string, error) {
	var combinedNonce, saltB64 string
	iterations := 0
	for _, part := range strings.Split(serverFirst, ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		switch key {
		case "r":
			combinedNonce = value
		case "s":
			saltB64 = value
		case "i":
			n, err := strconv.Atoi(value)
			if err != nil {
				return "", fmt.Errorf("invalid SCRAM iteration count %q", value)
			}
			iterations = n
		case "m":
			return "", fmt.Errorf("server requested an unsupported SCRAM extension")
		}
	}
	// Sem este teste o servidor escolheria o nonce sozinho e o desafio
	// deixaria de ser novo a cada conexao.
	if combinedNonce == "" || !strings.HasPrefix(combinedNonce, s.nonce) {
		return "", fmt.Errorf("server SCRAM nonce does not extend the client nonce")
	}
	if iterations <= 0 {
		return "", fmt.Errorf("invalid SCRAM iteration count")
	}
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return "", fmt.Errorf("invalid SCRAM salt: %w", err)
	}

	saltedPassword := pbkdf2SHA256([]byte(s.password), salt, iterations, sha256.Size)
	clientKey := hmacSHA256(saltedPassword, []byte("Client Key"))
	storedKey := sha256.Sum256(clientKey)
	s.serverKey = hmacSHA256(saltedPassword, []byte("Server Key"))

	// "biws" e base64("n,,"): o gs2-header repetido, como manda o RFC.
	withoutProof := "c=biws,r=" + combinedNonce
	s.authMessage = s.firstBare + "," + serverFirst + "," + withoutProof

	clientSignature := hmacSHA256(storedKey[:], []byte(s.authMessage))
	proof := make([]byte, len(clientKey))
	for i := range clientKey {
		proof[i] = clientKey[i] ^ clientSignature[i]
	}
	return withoutProof + ",p=" + base64.StdEncoding.EncodeToString(proof), nil
}

// verify confere a assinatura do servidor. Sem esta checagem a autenticacao
// deixa de ser mutua e um intermediario poderia se passar pelo banco.
func (s *scramClient) verify(serverFinal string) error {
	for _, part := range strings.Split(serverFinal, ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		switch key {
		case "e":
			return fmt.Errorf("server rejected SCRAM authentication: %s", value)
		case "v":
			signature, err := base64.StdEncoding.DecodeString(value)
			if err != nil {
				return fmt.Errorf("invalid SCRAM server signature: %w", err)
			}
			expected := hmacSHA256(s.serverKey, []byte(s.authMessage))
			if subtle.ConstantTimeCompare(signature, expected) != 1 {
				return fmt.Errorf("SCRAM server signature mismatch: the server could not prove it knows the password")
			}
			return nil
		}
	}
	return fmt.Errorf("server did not send a SCRAM signature")
}
