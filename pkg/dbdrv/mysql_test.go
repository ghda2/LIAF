package dbdrv

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"strings"
	"testing"
)

// O protocolo do MySQL nao tem um servidor falso aqui: o handshake completo
// com caching_sha2_password precisaria de chave RSA e de um dialeto de OK/EOF
// grande demais para caber num teste legivel. O que se testa e cada peca que
// o cliente monta ou interpreta — que e onde os erros de byte acontecem.

func TestMySQLLengthEncodedRoundTrip(t *testing.T) {
	for _, want := range []uint64{0, 1, 250, 251, 65535, 65536, 1 << 23, 1 << 24, 1 << 40} {
		encoded := myAppendLenenc(nil, want)
		got, rest := myReadLenenc(encoded)
		if got != want {
			t.Errorf("%d codificou/decodificou como %d", want, got)
		}
		if len(rest) != 0 {
			t.Errorf("%d deixou %d bytes sobrando", want, len(rest))
		}
	}
}

func TestMySQLLengthEncodedStringHandlesNull(t *testing.T) {
	s, rest, ok := myReadLenencString([]byte{0xfb, 'x'})
	if !ok || s != "" || len(rest) != 1 {
		t.Fatalf("0xfb devia ser NULL: s=%q rest=%v ok=%v", s, rest, ok)
	}
	body := append(myAppendLenenc(nil, 5), "alice"...)
	s, rest, ok = myReadLenencString(body)
	if !ok || s != "alice" || len(rest) != 0 {
		t.Fatalf("string: s=%q rest=%v ok=%v", s, rest, ok)
	}
	// Um pacote truncado nao pode virar panic de indice.
	if _, _, ok := myReadLenencString(append(myAppendLenenc(nil, 99), "curto"...)); ok {
		t.Fatal("string truncada foi aceita")
	}
}

// buildGreeting monta um Initial Handshake Packet v10 como o servidor envia.
func buildGreeting(version, plugin string, salt []byte, caps uint32) []byte {
	p := []byte{10}
	p = append(p, version...)
	p = append(p, 0)
	p = binary.LittleEndian.AppendUint32(p, 7) // connection id
	p = append(p, salt[:8]...)
	p = append(p, 0)                                            // filler
	p = binary.LittleEndian.AppendUint16(p, uint16(caps&0xffff)) // capacidades baixas
	p = append(p, myCharsetUTF8)
	p = binary.LittleEndian.AppendUint16(p, 2)             // status flags
	p = binary.LittleEndian.AppendUint16(p, uint16(caps>>16)) // capacidades altas
	p = append(p, byte(len(salt)+1))                       // tamanho do scramble
	p = append(p, make([]byte, 10)...)                     // reservado
	p = append(p, salt[8:]...)
	p = append(p, 0)
	p = append(p, plugin...)
	return append(p, 0)
}

func TestMySQLParseGreeting(t *testing.T) {
	salt := []byte("0123456789abcdefghijk") // 21 bytes, como o servidor manda
	caps := uint32(myCapProtocol41 | myCapPluginAuth | myCapSSL | myCapDeprecateEOF)
	gotSalt, plugin, gotCaps, err := parseGreeting(buildGreeting("8.0.36", "caching_sha2_password", salt, caps))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotSalt, salt) {
		t.Errorf("salt: %q, esperado %q", gotSalt, salt)
	}
	if plugin != "caching_sha2_password" {
		t.Errorf("plugin: %q", plugin)
	}
	if gotCaps&myCapSSL == 0 || gotCaps&myCapDeprecateEOF == 0 {
		t.Errorf("capacidades perdidas: %08x", gotCaps)
	}
}

func TestMySQLParseGreetingRejectsGarbage(t *testing.T) {
	if _, _, _, err := parseGreeting([]byte{9, 'x', 0}); err == nil {
		t.Error("versao de protocolo diferente de 10 foi aceita")
	}
	if _, _, _, err := parseGreeting(nil); err == nil {
		t.Error("pacote vazio foi aceito")
	}
	if _, _, _, err := parseGreeting([]byte{10, 'x', 0, 1, 2}); err == nil {
		t.Error("pacote truncado foi aceito")
	}
}

func TestMySQLNativePasswordMatchesReference(t *testing.T) {
	salt := []byte("abcdefghijklmnopqrst")
	got, err := myAuthResponse("mysql_native_password", "pencil", salt)
	if err != nil {
		t.Fatal(err)
	}
	// SHA1(senha) XOR SHA1(salt + SHA1(SHA1(senha)))
	first := sha1.Sum([]byte("pencil"))
	second := sha1.Sum(first[:])
	mac := sha1.New()
	mac.Write(salt)
	mac.Write(second[:])
	mask := mac.Sum(nil)
	want := make([]byte, len(first))
	for i := range first {
		want[i] = first[i] ^ mask[i]
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("resposta divergente\n got: %x\nwant: %x", got, want)
	}
	// Senha vazia nao envia prova nenhuma.
	if empty, _ := myAuthResponse("mysql_native_password", "", salt); len(empty) != 0 {
		t.Fatalf("senha vazia produziu %d bytes", len(empty))
	}
}

func TestMySQLCachingSHA2MatchesReference(t *testing.T) {
	salt := []byte("abcdefghijklmnopqrst")
	got, err := myAuthResponse("caching_sha2_password", "pencil", salt)
	if err != nil {
		t.Fatal(err)
	}
	first := sha256.Sum256([]byte("pencil"))
	second := sha256.Sum256(first[:])
	mac := sha256.New()
	mac.Write(second[:])
	mac.Write(salt)
	mask := mac.Sum(nil)
	want := make([]byte, len(first))
	for i := range first {
		want[i] = first[i] ^ mask[i]
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("resposta divergente\n got: %x\nwant: %x", got, want)
	}
}

func TestMySQLRejectsUnknownAuthPlugin(t *testing.T) {
	if _, err := myAuthResponse("sha256_password_v9", "x", []byte("salt")); err == nil {
		t.Fatal("plugin desconhecido foi aceito silenciosamente")
	}
}

func TestMySQLExecutePacketKeepsArgumentsOutOfSQL(t *testing.T) {
	hostile := "x'; DROP TABLE users; --"
	packet := myExecutePacket(9, []Value{int64(7), hostile, nil, 2.5, true})

	if packet[0] != myComStmtExec {
		t.Fatalf("comando errado: 0x%02x", packet[0])
	}
	if binary.LittleEndian.Uint32(packet[1:5]) != 9 {
		t.Error("statement id nao foi preservado")
	}
	// O valor hostil viaja como dado tipado; nada aqui e texto de comando.
	if !bytes.Contains(packet, []byte(hostile)) {
		t.Fatal("o argumento nao foi enviado")
	}
	// Mapa de nulos: so o terceiro parametro (indice 2) e NULL.
	nullMask := packet[10]
	if nullMask != 0b00000100 {
		t.Errorf("mapa de nulos: %08b", nullMask)
	}
	if packet[11] != 1 {
		t.Error("new-params-bound-flag nao foi ligado")
	}
	types := packet[12:22]
	want := []byte{myTypeLongLong, 0, myTypeVarString, 0, myTypeNull, 0, myTypeDouble, 0, myTypeTiny, 0}
	if !bytes.Equal(types, want) {
		t.Errorf("tipos enviados: %v, esperado %v", types, want)
	}
}

func TestMySQLExecutePacketWithoutArguments(t *testing.T) {
	packet := myExecutePacket(3, nil)
	if len(packet) != 10 {
		t.Fatalf("sem parametros o pacote tem 10 bytes, recebido %d", len(packet))
	}
}

func TestMySQLDecodeBinaryValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		col  myColumn
		body []byte
		want Value
	}{
		{"tinyint(1) vira bool", myColumn{kind: myTypeTiny, length: 1}, []byte{1}, true},
		{"tinyint largo vira int", myColumn{kind: myTypeTiny, length: 4}, []byte{255}, int64(-1)},
		{"tinyint unsigned", myColumn{kind: myTypeTiny, length: 4, unsigned: true}, []byte{255}, uint64(255)},
		{"int", myColumn{kind: myTypeLong}, []byte{42, 0, 0, 0}, int64(42)},
		{"bigint", myColumn{kind: myTypeLongLong}, []byte{7, 0, 0, 0, 0, 0, 0, 0}, int64(7)},
		{"texto", myColumn{kind: myTypeVarString}, append([]byte{5}, "alice"...), "alice"},
		{"decimal vira float", myColumn{kind: myTypeNewDecimal}, append([]byte{4}, "9.50"...), 9.5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, rest, err := myDecodeBinaryValue(tc.col, tc.body)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("%#v, esperado %#v", got, tc.want)
			}
			if len(rest) != 0 {
				t.Fatalf("sobraram %d bytes", len(rest))
			}
		})
	}
}

func TestMySQLDecodeDateAndTime(t *testing.T) {
	date := []byte{4, 0xe8, 0x07, 9, 16} // 2024-09-16 em 4 bytes
	got, _, err := myDecodeDate(date)
	if err != nil || got != "2024-09-16" {
		t.Errorf("data: %v (%v)", got, err)
	}
	datetime := []byte{7, 0xe8, 0x07, 9, 16, 13, 45, 30}
	got, _, err = myDecodeDate(datetime)
	if err != nil || got != "2024-09-16T13:45:30" {
		t.Errorf("datetime: %v (%v)", got, err)
	}
	clock := []byte{8, 0, 1, 0, 0, 0, 2, 30, 0} // 1 dia, 02:30:00
	got, _, err = myDecodeTime(clock)
	if err != nil || got != "26:30:00" {
		t.Errorf("time: %v (%v)", got, err)
	}
}

func TestMySQLDecodeBinaryRowNullBitmap(t *testing.T) {
	columns := []myColumn{{name: "id", kind: myTypeLong}, {name: "name", kind: myTypeVarString}}
	// Cabecalho 0x00, mapa de nulos com a coluna 1 (offset 3) marcada.
	packet := []byte{0x00, 0b00001000, 42, 0, 0, 0}
	values, err := myDecodeBinaryRow(packet, columns)
	if err != nil {
		t.Fatal(err)
	}
	if values[0] != int64(42) {
		t.Errorf("id: %#v", values[0])
	}
	if values[1] != nil {
		t.Errorf("name devia ser NULL, recebido %#v", values[1])
	}
}

func TestMySQLErrorCarriesCodeAndState(t *testing.T) {
	packet := []byte{myErr, 0x46, 0x04} // 1094
	packet = append(packet, "#HY000Unknown thread id"...)
	err := myError(packet)
	if err == nil {
		t.Fatal("pacote de erro virou sucesso")
	}
	for _, want := range []string{"1094", "HY000", "Unknown thread id"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("mensagem sem %q: %v", want, err)
		}
	}
}
