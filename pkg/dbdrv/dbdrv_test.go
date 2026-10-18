package dbdrv

import (
	"strings"
	"testing"
)

func TestParseDSNURLForm(t *testing.T) {
	cfg, err := parseDSN("postgres://app:s3%40cr3t@db.internal:6543/loja?sslmode=require&pool_max=4", DriverPostgres, "5432")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.host != "db.internal" || cfg.port != "6543" {
		t.Errorf("endereco: %s", cfg.address())
	}
	if cfg.user != "app" {
		t.Errorf("user: %q", cfg.user)
	}
	// A senha vem percent-decoded: um @ na senha e comum e nao pode partir o DSN.
	if cfg.password != "s3@cr3t" {
		t.Errorf("password: %q", cfg.password)
	}
	if cfg.database != "loja" {
		t.Errorf("database: %q", cfg.database)
	}
	if cfg.param("sslmode", "") != "require" {
		t.Errorf("sslmode: %q", cfg.param("sslmode", ""))
	}
}

func TestParseDSNDefaults(t *testing.T) {
	cfg, err := parseDSN("redis://localhost", DriverRedis, "6379")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.port != "6379" {
		t.Errorf("porta padrao nao aplicada: %q", cfg.port)
	}
	if got := cfg.param("ausente", "padrao"); got != "padrao" {
		t.Errorf("fallback do parametro: %q", got)
	}
}

func TestParseDSNKeywordFormIsPostgresOnly(t *testing.T) {
	cfg, err := parseDSN("host=db.internal port=6543 user=app password='uma senha' dbname=loja sslmode=disable", DriverPostgres, "5432")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.host != "db.internal" || cfg.port != "6543" || cfg.user != "app" || cfg.database != "loja" {
		t.Errorf("campos: %+v", cfg)
	}
	// Aspas simples delimitam valor com espaco, como no libpq.
	if cfg.password != "uma senha" {
		t.Errorf("password: %q", cfg.password)
	}
	if cfg.param("sslmode", "") != "disable" {
		t.Errorf("sslmode: %q", cfg.param("sslmode", ""))
	}

	// A forma de palavras-chave nao existe no MySQL nem no Redis; aceita-la
	// ali produziria uma conexao a localhost sem o usuario pedido.
	if _, err := parseDSN("host=x user=y", DriverMySQL, "3306"); err == nil {
		t.Error("forma de palavras-chave foi aceita no mysql")
	}
}

func TestParseDSNRejectsEmpty(t *testing.T) {
	if _, err := parseDSN("   ", DriverPostgres, "5432"); err == nil {
		t.Fatal("DSN vazio foi aceito")
	}
}

func TestOpenRejectsUnknownDriver(t *testing.T) {
	_, err := Open("postgresql", "postgres://x@localhost/db")
	if err == nil {
		t.Fatal("driver desconhecido foi aceito")
	}
	// A mensagem precisa listar o que existe: "postgresql" e um erro comum.
	for _, want := range Drivers {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("mensagem nao cita %q: %v", want, err)
		}
	}
}

func TestTextValueCoversLIAFScalars(t *testing.T) {
	for _, tc := range []struct {
		in    Value
		want  string
		valid bool
	}{
		{nil, "", false}, // NULL nao tem texto
		{true, "t", true},
		{false, "f", true},
		{int64(-7), "-7", true},
		{7, "7", true}, // literal do fonte chega como int, nao int64
		{uint64(18446744073709551615), "18446744073709551615", true},
		{2.5, "2.5", true},
		{"alice", "alice", true},
	} {
		got, ok := textValue(tc.in)
		if ok != tc.valid || (tc.valid && got != tc.want) {
			t.Errorf("%#v: got %q/%v, want %q/%v", tc.in, got, ok, tc.want, tc.valid)
		}
	}
}
