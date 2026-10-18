package dbdrv

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

const (
	myOK           = 0x00
	myEOF          = 0xfe
	myErr          = 0xff
	myAuthMoreData = 0x01

	myMaxPacketSize = 1<<24 - 1
)

func (c *mysqlConn) readOK() ([]byte, error) {
	p, err := c.read()
	if err != nil {
		return nil, err
	}
	if len(p) > 0 && p[0] == myErr {
		return nil, myError(p)
	}
	return p, nil
}

// write divide a carga em pacotes de no maximo 16 MiB - 1, como manda o
// protocolo, e mantem o numero de sequencia.
func (c *mysqlConn) write(payload []byte) error {
	for {
		chunk := payload
		if len(chunk) > myMaxPacketSize {
			chunk = chunk[:myMaxPacketSize]
		}
		header := []byte{byte(len(chunk)), byte(len(chunk) >> 8), byte(len(chunk) >> 16), c.sequence}
		c.sequence++
		if _, err := c.w.Write(header); err != nil {
			return err
		}
		if _, err := c.w.Write(chunk); err != nil {
			return err
		}
		payload = payload[len(chunk):]
		// Uma carga de exatamente 16 MiB - 1 exige um pacote vazio a seguir
		// para o servidor saber que acabou.
		if len(chunk) < myMaxPacketSize {
			break
		}
	}
	return c.w.Flush()
}

func (c *mysqlConn) read() ([]byte, error) {
	var payload []byte
	for {
		header := make([]byte, 4)
		if _, err := io.ReadFull(c.r, header); err != nil {
			return nil, fmt.Errorf("%w: reading from mysql: %v", ErrConnLost, err)
		}
		size := int(header[0]) | int(header[1])<<8 | int(header[2])<<16
		c.sequence = header[3] + 1
		chunk := make([]byte, size)
		if _, err := io.ReadFull(c.r, chunk); err != nil {
			return nil, fmt.Errorf("%w: reading from mysql: %v", ErrConnLost, err)
		}
		payload = append(payload, chunk...)
		if size < myMaxPacketSize {
			return payload, nil
		}
	}
}

func myAppendLenenc(dst []byte, n uint64) []byte {
	switch {
	case n < 251:
		return append(dst, byte(n))
	case n < 1<<16:
		return binary.LittleEndian.AppendUint16(append(dst, 0xfc), uint16(n))
	case n < 1<<24:
		return append(append(dst, 0xfd), byte(n), byte(n>>8), byte(n>>16))
	default:
		return binary.LittleEndian.AppendUint64(append(dst, 0xfe), n)
	}
}

func myReadLenenc(b []byte) (uint64, []byte) {
	if len(b) == 0 {
		return 0, b
	}
	switch b[0] {
	case 0xfc:
		if len(b) < 3 {
			return 0, nil
		}
		return uint64(binary.LittleEndian.Uint16(b[1:3])), b[3:]
	case 0xfd:
		if len(b) < 4 {
			return 0, nil
		}
		return uint64(b[1]) | uint64(b[2])<<8 | uint64(b[3])<<16, b[4:]
	case 0xfe:
		if len(b) < 9 {
			return 0, nil
		}
		return binary.LittleEndian.Uint64(b[1:9]), b[9:]
	default:
		return uint64(b[0]), b[1:]
	}
}

func myReadLenencString(b []byte) (string, []byte, bool) {
	if len(b) > 0 && b[0] == 0xfb { // NULL
		return "", b[1:], true
	}
	n, rest := myReadLenenc(b)
	if rest == nil || uint64(len(rest)) < n {
		return "", nil, false
	}
	return string(rest[:n]), rest[n:], true
}

// myError traduz o ERR_Packet: codigo(2), marcador '#' + SQLSTATE(5), mensagem.
func myError(p []byte) error {
	if len(p) < 3 {
		return fmt.Errorf("mysql returned an unreadable error")
	}
	code := binary.LittleEndian.Uint16(p[1:3])
	rest := p[3:]
	state := ""
	if len(rest) > 6 && rest[0] == '#' {
		state = string(rest[1:6])
		rest = rest[6:]
	}
	message := strings.TrimRight(string(rest), "\x00")
	if state != "" {
		return fmt.Errorf("mysql error %d (SQLSTATE %s): %s", code, state, message)
	}
	return fmt.Errorf("mysql error %d: %s", code, message)
}
