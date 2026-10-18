package dbdrv

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const (
	pgProtocolVersion = 196608 // 3.0 em (major<<16 | minor)
	pgSSLRequestCode  = 80877103

	pgAuthOK                = 0
	pgAuthCleartextPassword = 3
	pgAuthMD5Password       = 5
	pgAuthSASL              = 10
	pgAuthSASLContinue      = 11
	pgAuthSASLFinal         = 12
)

// pgNegotiateTLS envia o SSLRequest antes do StartupMessage. O servidor
// responde com um unico byte: 'S' aceita, 'N' recusa.
func pgNegotiateTLS(raw net.Conn, host, sslmode string) (net.Conn, error) {
	switch sslmode {
	case "disable":
		return raw, nil
	case "prefer", "allow", "require", "verify-ca", "verify-full":
	default:
		raw.Close()
		return nil, fmt.Errorf("unsupported sslmode %q", sslmode)
	}

	_ = raw.SetDeadline(time.Now().Add(ioTimeout))
	request := make([]byte, 8)
	binary.BigEndian.PutUint32(request[0:4], 8)
	binary.BigEndian.PutUint32(request[4:8], pgSSLRequestCode)
	if _, err := raw.Write(request); err != nil {
		raw.Close()
		return nil, fmt.Errorf("could not send SSLRequest: %w", err)
	}
	answer := make([]byte, 1)
	if _, err := io.ReadFull(raw, answer); err != nil {
		raw.Close()
		return nil, fmt.Errorf("could not read the SSLRequest reply: %w", err)
	}
	if answer[0] != 'S' {
		if sslmode == "prefer" || sslmode == "allow" {
			return raw, nil
		}
		raw.Close()
		return nil, fmt.Errorf("server refused TLS but sslmode=%s requires it", sslmode)
	}

	// prefer e allow cifram o trafego mas nao autenticam o servidor: nesses
	// modos o libpq tambem nao verifica, e exigir verificacao aqui quebraria
	// o caso comum do certificado autoassinado em desenvolvimento.
	conf := &tls.Config{ServerName: host}
	if sslmode != "verify-ca" && sslmode != "verify-full" {
		conf.InsecureSkipVerify = true
	}
	tlsConn := tls.Client(raw, conf)
	if err := tlsConn.Handshake(); err != nil {
		raw.Close()
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}
	return tlsConn, nil
}

func (c *pgConn) startup(cfg *config) error {
	_ = c.conn.SetDeadline(time.Now().Add(ioTimeout))

	var body []byte
	body = binary.BigEndian.AppendUint32(body, pgProtocolVersion)
	body = pgAppendString(body, "user")
	body = pgAppendString(body, cfg.user)
	if cfg.database != "" {
		body = pgAppendString(body, "database")
		body = pgAppendString(body, cfg.database)
	}
	if app := cfg.param("application_name", "liaf"); app != "" {
		body = pgAppendString(body, "application_name")
		body = pgAppendString(body, app)
	}
	body = append(body, 0)

	// O StartupMessage e a unica mensagem sem byte de tipo.
	frame := binary.BigEndian.AppendUint32(nil, uint32(len(body)+4))
	if _, err := c.w.Write(append(frame, body...)); err != nil {
		return err
	}
	if err := c.w.Flush(); err != nil {
		return err
	}
	return c.authenticate(cfg)
}

func (c *pgConn) authenticate(cfg *config) error {
	var scram *scramClient
	for {
		kind, payload, err := c.receive()
		if err != nil {
			return err
		}
		switch kind {
		case 'R':
			if len(payload) < 4 {
				return fmt.Errorf("malformed authentication message")
			}
			code := binary.BigEndian.Uint32(payload[:4])
			rest := payload[4:]
			switch code {
			case pgAuthOK:
			case pgAuthCleartextPassword:
				if err := c.send('p', pgAppendString(nil, cfg.password)); err != nil {
					return err
				}
			case pgAuthMD5Password:
				if len(rest) < 4 {
					return fmt.Errorf("malformed MD5 authentication salt")
				}
				if err := c.send('p', pgAppendString(nil, pgMD5Password(cfg.user, cfg.password, rest[:4]))); err != nil {
					return err
				}
			case pgAuthSASL:
				if !pgOffersSCRAM(rest) {
					return fmt.Errorf("server offered no supported SASL mechanism (only SCRAM-SHA-256 is implemented)")
				}
				scram, err = newSCRAMClient(cfg.password)
				if err != nil {
					return err
				}
				first := scram.first()
				out := pgAppendString(nil, "SCRAM-SHA-256")
				out = binary.BigEndian.AppendUint32(out, uint32(len(first)))
				out = append(out, first...)
				if err := c.send('p', out); err != nil {
					return err
				}
			case pgAuthSASLContinue:
				if scram == nil {
					return fmt.Errorf("unexpected SASLContinue before SASL start")
				}
				final, err := scram.final(string(rest))
				if err != nil {
					return err
				}
				if err := c.send('p', []byte(final)); err != nil {
					return err
				}
			case pgAuthSASLFinal:
				if scram == nil {
					return fmt.Errorf("unexpected SASLFinal before SASL start")
				}
				if err := scram.verify(string(rest)); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported authentication method %d", code)
			}
		case 'S', 'K', 'N':
			// ParameterStatus, BackendKeyData e NoticeResponse nao mudam o
			// que a LIAF expoe; o cancelamento assincrono nao e suportado.
		case 'Z':
			return nil
		case 'E':
			return pgError(payload)
		default:
			return fmt.Errorf("unexpected message %q during handshake", string(kind))
		}
	}
}

func pgOffersSCRAM(payload []byte) bool {
	for _, mechanism := range strings.Split(string(payload), "\x00") {
		if mechanism == "SCRAM-SHA-256" {
			return true
		}
	}
	return false
}

// pgMD5Password produz "md5" + hex(md5(hex(md5(senha+usuario)) + salt)).
func pgMD5Password(user, password string, salt []byte) string {
	inner := md5.Sum([]byte(password + user))
	outer := md5.Sum(append([]byte(hex.EncodeToString(inner[:])), salt...))
	return "md5" + hex.EncodeToString(outer[:])
}
