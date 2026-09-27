package dbdrv

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
)

// Tipos de coluna do protocolo binario.
const (
	myTypeDecimal    = 0x00
	myTypeTiny       = 0x01
	myTypeShort      = 0x02
	myTypeLong       = 0x03
	myTypeFloat      = 0x04
	myTypeDouble     = 0x05
	myTypeNull       = 0x06
	myTypeTimestamp  = 0x07
	myTypeLongLong   = 0x08
	myTypeInt24      = 0x09
	myTypeDate       = 0x0a
	myTypeTime       = 0x0b
	myTypeDatetime   = 0x0c
	myTypeYear       = 0x0d
	myTypeNewDecimal = 0xf6
	myTypeVarString  = 0xfd
	myTypeString     = 0xfe

	myFlagUnsigned = 0x0020
)

type myColumn struct {
	name     string
	kind     byte
	unsigned bool
	length   uint32
}

func (c *mysqlConn) readColumns(n int) ([]myColumn, error) {
	columns := make([]myColumn, 0, n)
	for i := 0; i < n; i++ {
		p, err := c.read()
		if err != nil {
			return nil, err
		}
		if len(p) > 0 && p[0] == myErr {
			return nil, myError(p)
		}
		col, err := parseColumn(p)
		if err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}
	if n > 0 && c.caps&myCapDeprecateEOF == 0 {
		if _, err := c.read(); err != nil { // EOF packet
			return nil, err
		}
	}
	return columns, nil
}

// parseColumn le o ColumnDefinition41: seis strings de tamanho variavel
// (catalog, schema, tabela, tabela original, nome, nome original) seguidas
// dos metadados de tipo.
func parseColumn(p []byte) (myColumn, error) {
	rest := p
	var name string
	for i := 0; i < 6; i++ {
		s, tail, ok := myReadLenencString(rest)
		if !ok {
			return myColumn{}, fmt.Errorf("malformed column definition")
		}
		if i == 4 {
			name = s
		}
		rest = tail
	}
	if len(rest) < 1+2+4+1+2 {
		return myColumn{}, fmt.Errorf("truncated column definition")
	}
	rest = rest[1+2:] // tamanho dos campos fixos + charset
	length := binary.LittleEndian.Uint32(rest[:4])
	kind := rest[4]
	flags := binary.LittleEndian.Uint16(rest[5:7])
	return myColumn{name: name, kind: kind, unsigned: flags&myFlagUnsigned != 0, length: length}, nil
}

// myExecutePacket monta o COM_STMT_EXECUTE. Os parametros vao num bloco
// tipado, separado do texto preparado: nenhum valor pode virar sintaxe.
func myExecutePacket(stmtID uint32, args []Value) []byte {
	out := binary.LittleEndian.AppendUint32([]byte{myComStmtExec}, stmtID)
	out = append(out, 0)                           // CURSOR_TYPE_NO_CURSOR
	out = binary.LittleEndian.AppendUint32(out, 1) // iteration count
	if len(args) == 0 {
		return out
	}

	nullMask := make([]byte, (len(args)+7)/8)
	for i, a := range args {
		if a == nil {
			nullMask[i/8] |= 1 << (i % 8)
		}
	}
	out = append(out, nullMask...)
	out = append(out, 1) // new-params-bound-flag

	var values []byte
	for _, a := range args {
		switch v := a.(type) {
		case nil:
			out = append(out, myTypeNull, 0)
		case bool:
			out = append(out, myTypeTiny, 0)
			if v {
				values = append(values, 1)
			} else {
				values = append(values, 0)
			}
		case int64:
			out = append(out, myTypeLongLong, 0)
			values = binary.LittleEndian.AppendUint64(values, uint64(v))
		case int:
			// Literal inteiro do fonte LIAF: chega como int, nao como int64.
			out = append(out, myTypeLongLong, 0)
			values = binary.LittleEndian.AppendUint64(values, uint64(v))
		case uint64:
			out = append(out, myTypeLongLong, 0x80)
			values = binary.LittleEndian.AppendUint64(values, v)
		case float64:
			out = append(out, myTypeDouble, 0)
			values = binary.LittleEndian.AppendUint64(values, math.Float64bits(v))
		default:
			text, _ := textValue(v)
			out = append(out, myTypeVarString, 0)
			values = append(myAppendLenenc(values, uint64(len(text))), text...)
		}
	}
	return append(out, values...)
}

func (c *mysqlConn) readBinaryRows(columns []myColumn) (*Rows, error) {
	rows := &Rows{Columns: make([]string, len(columns))}
	for i, col := range columns {
		rows.Columns[i] = col.name
	}
	for {
		p, err := c.read()
		if err != nil {
			return nil, err
		}
		if len(p) == 0 {
			return nil, fmt.Errorf("empty row packet")
		}
		if p[0] == myErr {
			return nil, myError(p)
		}
		// Um pacote curto comecando em 0xfe encerra o result set — em 0x00
		// quando o servidor negociou DEPRECATE_EOF, e ai o OK vem no lugar.
		if p[0] == myEOF && len(p) < 9 {
			return rows, nil
		}
		if p[0] == myOK && c.caps&myCapDeprecateEOF != 0 && len(p) < 9 && len(columns) == 0 {
			return rows, nil
		}
		values, err := myDecodeBinaryRow(p, columns)
		if err != nil {
			return nil, err
		}
		rows.Values = append(rows.Values, values)
	}
}

// myDecodeBinaryRow le uma linha do protocolo binario: cabecalho 0x00, mapa
// de nulos deslocado em 2 bits, e os valores em sequencia.
func myDecodeBinaryRow(p []byte, columns []myColumn) ([]Value, error) {
	maskLen := (len(columns) + 9) / 8
	if len(p) < 1+maskLen {
		return nil, fmt.Errorf("truncated binary row")
	}
	mask := p[1 : 1+maskLen]
	rest := p[1+maskLen:]
	values := make([]Value, 0, len(columns))
	for i, col := range columns {
		offset := i + 2 // os dois primeiros bits do mapa sao reservados
		if mask[offset/8]&(1<<(offset%8)) != 0 {
			values = append(values, nil)
			continue
		}
		v, tail, err := myDecodeBinaryValue(col, rest)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
		rest = tail
	}
	return values, nil
}

func myDecodeBinaryValue(col myColumn, b []byte) (Value, []byte, error) {
	need := func(n int) error {
		if len(b) < n {
			return fmt.Errorf("truncated value for column %q", col.name)
		}
		return nil
	}
	switch col.kind {
	case myTypeTiny:
		if err := need(1); err != nil {
			return nil, nil, err
		}
		// TINYINT(1) e como o MySQL guarda BOOLEAN.
		if col.length == 1 && !col.unsigned {
			return b[0] != 0, b[1:], nil
		}
		if col.unsigned {
			return uint64(b[0]), b[1:], nil
		}
		return int64(int8(b[0])), b[1:], nil
	case myTypeShort, myTypeYear:
		if err := need(2); err != nil {
			return nil, nil, err
		}
		raw := binary.LittleEndian.Uint16(b[:2])
		if col.unsigned {
			return uint64(raw), b[2:], nil
		}
		return int64(int16(raw)), b[2:], nil
	case myTypeLong, myTypeInt24:
		if err := need(4); err != nil {
			return nil, nil, err
		}
		raw := binary.LittleEndian.Uint32(b[:4])
		if col.unsigned {
			return uint64(raw), b[4:], nil
		}
		return int64(int32(raw)), b[4:], nil
	case myTypeLongLong:
		if err := need(8); err != nil {
			return nil, nil, err
		}
		raw := binary.LittleEndian.Uint64(b[:8])
		if col.unsigned {
			return raw, b[8:], nil
		}
		return int64(raw), b[8:], nil
	case myTypeFloat:
		if err := need(4); err != nil {
			return nil, nil, err
		}
		return float64(math.Float32frombits(binary.LittleEndian.Uint32(b[:4]))), b[4:], nil
	case myTypeDouble:
		if err := need(8); err != nil {
			return nil, nil, err
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(b[:8])), b[8:], nil
	case myTypeDate, myTypeDatetime, myTypeTimestamp:
		return myDecodeDate(b)
	case myTypeTime:
		return myDecodeTime(b)
	case myTypeDecimal, myTypeNewDecimal:
		s, tail, ok := myReadLenencString(b)
		if !ok {
			return nil, nil, fmt.Errorf("truncated decimal for column %q", col.name)
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, tail, nil
		}
		return s, tail, nil
	default:
		s, tail, ok := myReadLenencString(b)
		if !ok {
			return nil, nil, fmt.Errorf("truncated value for column %q", col.name)
		}
		return s, tail, nil
	}
}

// myDecodeDate devolve a data em ISO-8601, que e o que json.Unmarshal aceita
// num campo str da struct LIAF.
func myDecodeDate(b []byte) (Value, []byte, error) {
	if len(b) < 1 {
		return nil, nil, fmt.Errorf("truncated date")
	}
	n := int(b[0])
	if len(b) < 1+n {
		return nil, nil, fmt.Errorf("truncated date")
	}
	body, tail := b[1:1+n], b[1+n:]
	switch {
	case n == 0:
		return "", tail, nil
	case n >= 11:
		micro := binary.LittleEndian.Uint32(body[7:11])
		return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d.%06d",
			binary.LittleEndian.Uint16(body[:2]), body[2], body[3], body[4], body[5], body[6], micro), tail, nil
	case n >= 7:
		return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d",
			binary.LittleEndian.Uint16(body[:2]), body[2], body[3], body[4], body[5], body[6]), tail, nil
	default:
		return fmt.Sprintf("%04d-%02d-%02d", binary.LittleEndian.Uint16(body[:2]), body[2], body[3]), tail, nil
	}
}

func myDecodeTime(b []byte) (Value, []byte, error) {
	if len(b) < 1 {
		return nil, nil, fmt.Errorf("truncated time")
	}
	n := int(b[0])
	if len(b) < 1+n {
		return nil, nil, fmt.Errorf("truncated time")
	}
	body, tail := b[1:1+n], b[1+n:]
	if n == 0 {
		return "00:00:00", tail, nil
	}
	sign := ""
	if body[0] == 1 {
		sign = "-"
	}
	days := binary.LittleEndian.Uint32(body[1:5])
	hours := uint32(body[5]) + days*24
	return fmt.Sprintf("%s%02d:%02d:%02d", sign, hours, body[6], body[7]), tail, nil
}
