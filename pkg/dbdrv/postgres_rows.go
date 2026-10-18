package dbdrv

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// OIDs dos tipos cujo texto convem converter antes de entregar a LIAF.
// Sem isso todo campo chegaria como string e um campo int da struct
// receberia "42" em vez de 42.
const (
	oidBool    = 16
	oidInt8    = 20
	oidInt2    = 21
	oidInt4    = 23
	oidFloat4  = 700
	oidFloat8  = 701
	oidNumeric = 1700
)

type pgColumn struct {
	name string
	oid  uint32
}

func pgRowDescription(payload []byte) []pgColumn {
	if len(payload) < 2 {
		return nil
	}
	count := int(binary.BigEndian.Uint16(payload[:2]))
	rest := payload[2:]
	columns := make([]pgColumn, 0, count)
	for i := 0; i < count; i++ {
		end := indexZero(rest)
		if end < 0 || len(rest) < end+1+18 {
			break
		}
		name := string(rest[:end])
		meta := rest[end+1:]
		// tableOID(4) columnIndex(2) typeOID(4) typeSize(2) modifier(4) format(2)
		columns = append(columns, pgColumn{name: name, oid: binary.BigEndian.Uint32(meta[6:10])})
		rest = meta[18:]
	}
	return columns
}

func pgDataRow(payload []byte, columns []pgColumn) ([]Value, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("malformed DataRow")
	}
	count := int(binary.BigEndian.Uint16(payload[:2]))
	rest := payload[2:]
	values := make([]Value, 0, count)
	for i := 0; i < count; i++ {
		if len(rest) < 4 {
			return nil, fmt.Errorf("truncated DataRow")
		}
		size := int32(binary.BigEndian.Uint32(rest[:4]))
		rest = rest[4:]
		if size < 0 {
			values = append(values, nil)
			continue
		}
		if len(rest) < int(size) {
			return nil, fmt.Errorf("truncated DataRow field")
		}
		text := string(rest[:size])
		rest = rest[size:]
		oid := uint32(0)
		if i < len(columns) {
			oid = columns[i].oid
		}
		values = append(values, pgDecodeText(oid, text))
	}
	return values, nil
}

// pgDecodeText traduz o texto do servidor no tipo Go correspondente. Tipos
// nao listados ficam como string: e o que a struct LIAF espera para texto,
// data, json, uuid e enum.
func pgDecodeText(oid uint32, text string) Value {
	switch oid {
	case oidBool:
		return text == "t" || text == "true"
	case oidInt2, oidInt4, oidInt8:
		if v, err := strconv.ParseInt(text, 10, 64); err == nil {
			return v
		}
	case oidFloat4, oidFloat8, oidNumeric:
		if v, err := strconv.ParseFloat(text, 64); err == nil {
			return v
		}
	}
	return text
}

// pgRowsAffected le a etiqueta do CommandComplete: "INSERT 0 1", "UPDATE 3",
// "DELETE 2", "SELECT 7". O ultimo campo e sempre a contagem.
func pgRowsAffected(payload []byte) int64 {
	tag := strings.TrimRight(string(payload), "\x00")
	fields := strings.Fields(tag)
	if len(fields) == 0 {
		return 0
	}
	n, err := strconv.ParseInt(fields[len(fields)-1], 10, 64)
	if err != nil {
		return 0
	}
	return n
}
