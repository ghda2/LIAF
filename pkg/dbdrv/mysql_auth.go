package dbdrv

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"time"
)

const (
	myCapLongPassword     = 0x00000001
	myCapFoundRows        = 0x00000002
	myCapLongFlag         = 0x00000004
	myCapConnectWithDB    = 0x00000008
	myCapLocalFiles       = 0x00000080
	myCapProtocol41       = 0x00000200
	myCapSSL              = 0x00000800
	myCapTransactions     = 0x00002000
	myCapSecureConnection = 0x00008000
	myCapPluginAuth       = 0x00080000
	myCapPluginAuthLenenc = 0x00200000
	myCapDeprecateEOF     = 0x01000000

	myCharsetUTF8 = 45 // utf8mb4_general_ci
)

func (c *mysqlConn) handshake(cfg *config) error {
	_ = c.conn.SetDeadline(time.Now().Add(ioTimeout))

	greeting, err := c.read()
	if err != nil {
		return fmt.Errorf("could not read the MySQL greeting: %w", err)
	}
	if len(greeting) > 0 && greeting[0] == myErr {
		return myError(greeting)
	}
	salt, plugin, serverCaps, err := parseGreeting(greeting)
	if err != nil {
		return err
	}

	c.caps = myCapLongPassword | myCapLongFlag | myCapProtocol41 | myCapTransactions |
		myCapSecureConnection | myCapPluginAuth | myCapPluginAuthLenenc | myCapFoundRows
	c.caps &^= myCapLocalFiles // LOAD DATA LOCAL le arquivos do cliente a pedido do servidor
	if serverCaps&myCapDeprecateEOF != 0 {
		c.caps |= myCapDeprecateEOF
	}
	if cfg.database != "" {
		c.caps |= myCapConnectWithDB
	}

	secure := false
	tlsMode := cfg.param("tls", "preferred")
	if tlsMode != "false" && tlsMode != "disable" && serverCaps&myCapSSL != 0 {
		c.caps |= myCapSSL
		if err := c.write(c.authHeader()); err != nil {
			return err
		}
		conf := &tls.Config{ServerName: cfg.host}
		if tlsMode != "verify" && tlsMode != "true" {
			conf.InsecureSkipVerify = true
		}
		tlsConn := tls.Client(c.conn, conf)
		if err := tlsConn.Handshake(); err != nil {
			return fmt.Errorf("TLS handshake failed: %w", err)
		}
		c.conn = tlsConn
		c.r = bufio.NewReader(tlsConn)
		c.w = bufio.NewWriter(tlsConn)
		secure = true
	} else if tlsMode == "true" || tlsMode == "verify" {
		return fmt.Errorf("server does not offer TLS but tls=%s requires it", tlsMode)
	}

	if plugin == "" {
		plugin = "mysql_native_password"
	}
	response, err := myAuthResponse(plugin, cfg.password, salt)
	if err != nil {
		return err
	}
	if err := c.write(c.authResponse(cfg, plugin, response)); err != nil {
		return err
	}
	return c.finishAuth(cfg, plugin, salt, secure)
}

// parseGreeting le o Initial Handshake Packet v10.
func parseGreeting(p []byte) (salt []byte, plugin string, caps uint32, err error) {
	if len(p) < 1 || p[0] != 10 {
		return nil, "", 0, fmt.Errorf("unsupported MySQL handshake (only protocol 10 is implemented)")
	}
	end := indexZero(p[1:])
	if end < 0 {
		return nil, "", 0, fmt.Errorf("malformed MySQL handshake")
	}
	rest := p[1+end+1:]
	if len(rest) < 8+1+2+1+2 {
		return nil, "", 0, fmt.Errorf("truncated MySQL handshake")
	}
	rest = rest[4:] // connection id
	salt = append(salt, rest[:8]...)
	rest = rest[8+1:] // scramble parte 1 + filler
	caps = uint32(binary.LittleEndian.Uint16(rest[:2]))
	rest = rest[2:]
	if len(rest) < 1 {
		return salt, "", caps, nil
	}
	if len(rest) < 1+2+2+1+10 {
		return nil, "", 0, fmt.Errorf("truncated MySQL handshake")
	}
	rest = rest[1+2:] // charset + status flags
	caps |= uint32(binary.LittleEndian.Uint16(rest[:2])) << 16
	rest = rest[2:]
	saltLen := int(rest[0])
	rest = rest[1+10:] // tamanho do scramble + reservado
	if saltLen > 8 {
		take := saltLen - 8
		if take > len(rest) {
			take = len(rest)
		}
		// O ultimo byte do scramble e um NUL terminador, nao entropia.
		salt = append(salt, bytes.TrimRight(rest[:take], "\x00")...)
		rest = rest[take:]
	}
	if caps&myCapPluginAuth != 0 && len(rest) > 0 {
		plugin = string(bytes.TrimRight(rest, "\x00"))
	}
	return salt, plugin, caps, nil
}

// authHeader e o prefixo de 32 bytes comum ao SSLRequest e ao
// HandshakeResponse41. Enviado sozinho, e o pedido de upgrade para TLS.
func (c *mysqlConn) authHeader() []byte {
	out := make([]byte, 0, 32)
	out = binary.LittleEndian.AppendUint32(out, c.caps)
	out = binary.LittleEndian.AppendUint32(out, myMaxPacketSize)
	out = append(out, myCharsetUTF8)
	return append(out, make([]byte, 23)...)
}

func (c *mysqlConn) authResponse(cfg *config, plugin string, auth []byte) []byte {
	out := c.authHeader()
	out = append(append(out, cfg.user...), 0)
	out = append(myAppendLenenc(out, uint64(len(auth))), auth...)
	if c.caps&myCapConnectWithDB != 0 {
		out = append(append(out, cfg.database...), 0)
	}
	if c.caps&myCapPluginAuth != 0 {
		out = append(append(out, plugin...), 0)
	}
	return out
}

// myAuthResponse calcula a prova de conhecimento da senha sem transmiti-la.
func myAuthResponse(plugin, password string, salt []byte) ([]byte, error) {
	if password == "" {
		return nil, nil
	}
	switch plugin {
	case "mysql_native_password":
		// SHA1(senha) XOR SHA1(salt + SHA1(SHA1(senha)))
		first := sha1.Sum([]byte(password))
		second := sha1.Sum(first[:])
		mac := sha1.New()
		mac.Write(salt)
		mac.Write(second[:])
		mask := mac.Sum(nil)
		out := make([]byte, len(first))
		for i := range first {
			out[i] = first[i] ^ mask[i]
		}
		return out, nil
	case "caching_sha2_password":
		// XOR(SHA256(senha), SHA256(SHA256(SHA256(senha)) + salt))
		first := sha256.Sum256([]byte(password))
		second := sha256.Sum256(first[:])
		mac := sha256.New()
		mac.Write(second[:])
		mac.Write(salt)
		mask := mac.Sum(nil)
		out := make([]byte, len(first))
		for i := range first {
			out[i] = first[i] ^ mask[i]
		}
		return out, nil
	case "mysql_clear_password":
		return append([]byte(password), 0), nil
	default:
		return nil, fmt.Errorf("unsupported MySQL authentication plugin %q", plugin)
	}
}

// finishAuth trata a troca extra que caching_sha2_password pode pedir e o
// AuthSwitchRequest, que o servidor manda quando o plugin adivinhado nao e o
// configurado para o usuario.
func (c *mysqlConn) finishAuth(cfg *config, plugin string, salt []byte, secure bool) error {
	for {
		p, err := c.read()
		if err != nil {
			return err
		}
		if len(p) == 0 {
			return fmt.Errorf("empty authentication packet")
		}
		switch p[0] {
		case myOK:
			return nil
		case myErr:
			return myError(p)
		case myEOF:
			// AuthSwitchRequest: plugin\0 + salt
			rest := p[1:]
			end := indexZero(rest)
			if end < 0 {
				return fmt.Errorf("malformed authentication switch request")
			}
			plugin = string(rest[:end])
			salt = bytes.TrimRight(rest[end+1:], "\x00")
			response, err := myAuthResponse(plugin, cfg.password, salt)
			if err != nil {
				return err
			}
			if err := c.write(response); err != nil {
				return err
			}
		case myAuthMoreData:
			if plugin != "caching_sha2_password" || len(p) < 2 {
				return fmt.Errorf("unexpected AuthMoreData from server")
			}
			switch p[1] {
			case 0x03: // fast auth: a senha ja estava no cache do servidor
			case 0x04: // full auth: o servidor precisa da senha em claro
				if err := c.fullSHA256Auth(cfg.password, salt, secure); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported caching_sha2_password state 0x%02x", p[1])
			}
		default:
			return fmt.Errorf("unexpected authentication packet 0x%02x", p[0])
		}
	}
}

// fullSHA256Auth executa o caminho lento do caching_sha2_password. Sobre TLS
// a senha vai em claro dentro do tunel; sem TLS ela e cifrada com a chave
// publica RSA do servidor, que nunca deve trafegar em claro.
func (c *mysqlConn) fullSHA256Auth(password string, salt []byte, secure bool) error {
	if secure {
		return c.write(append([]byte(password), 0))
	}
	if err := c.write([]byte{0x02}); err != nil { // request public key
		return err
	}
	p, err := c.read()
	if err != nil {
		return err
	}
	if len(p) < 2 || p[0] != myAuthMoreData {
		return fmt.Errorf("server did not send its RSA public key")
	}
	block, _ := pem.Decode(p[1:])
	if block == nil {
		return fmt.Errorf("could not decode the server RSA public key")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("invalid server RSA public key: %w", err)
	}
	pub, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("server public key is not RSA")
	}
	plain := append([]byte(password), 0)
	for i := range plain {
		plain[i] ^= salt[i%len(salt)]
	}
	cipher, err := rsa.EncryptOAEP(sha1.New(), rand.Reader, pub, plain, nil)
	if err != nil {
		return fmt.Errorf("could not encrypt the password: %w", err)
	}
	return c.write(cipher)
}
