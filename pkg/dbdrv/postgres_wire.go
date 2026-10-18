package dbdrv

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// Limite defensivo: uma mensagem corrompida nao deve virar uma alocacao de
// gigabytes antes de falhar.
const pgMaxMessageSize = 64 << 20

func indexZero(b []byte) int {
	for i, c := range b {
		if c == 0 {
			return i
		}
	}
	return -1
}

func pgAppendString(dst []byte, s string) []byte { return append(append(dst, s...), 0) }

// send escreve uma mensagem e ja faz flush: o protocolo e sincrono e o
// servidor so responde ao receber o Sync.
func (c *pgConn) send(kind byte, body []byte) error {
	c.buf = c.buf[:0]
	c.buf = append(c.buf, kind)
	c.buf = binary.BigEndian.AppendUint32(c.buf, uint32(len(body)+4))
	c.buf = append(c.buf, body...)
	if _, err := c.w.Write(c.buf); err != nil {
		return err
	}
	return c.w.Flush()
}

func (c *pgConn) receive() (byte, []byte, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(c.r, header); err != nil {
		return 0, nil, fmt.Errorf("%w: reading from postgres: %v", ErrConnLost, err)
	}
	size := int(binary.BigEndian.Uint32(header[1:5]))
	if size < 4 || size-4 > pgMaxMessageSize {
		return 0, nil, fmt.Errorf("postgres sent an invalid message length (%d)", size)
	}
	body := make([]byte, size-4)
	if _, err := io.ReadFull(c.r, body); err != nil {
		return 0, nil, fmt.Errorf("%w: reading from postgres: %v", ErrConnLost, err)
	}
	return header[0], body, nil
}

// pgError monta a mensagem a partir dos campos do ErrorResponse, priorizando
// severidade, codigo SQLSTATE e mensagem — que e o que ajuda a corrigir.
func pgError(payload []byte) error {
	fields := map[byte]string{}
	rest := payload
	for len(rest) > 0 && rest[0] != 0 {
		code := rest[0]
		end := indexZero(rest[1:])
		if end < 0 {
			break
		}
		fields[code] = string(rest[1 : 1+end])
		rest = rest[1+end+1:]
	}
	message := fields['M']
	if message == "" {
		message = "unknown error"
	}
	parts := []string{message}
	if code := fields['C']; code != "" {
		parts = append(parts, "SQLSTATE "+code)
	}
	if detail := fields['D']; detail != "" {
		parts = append(parts, detail)
	}
	if hint := fields['H']; hint != "" {
		parts = append(parts, "hint: "+hint)
	}
	severity := fields['S']
	if severity == "" {
		severity = "ERROR"
	}
	return fmt.Errorf("postgres %s: %s", severity, strings.Join(parts, " | "))
}
