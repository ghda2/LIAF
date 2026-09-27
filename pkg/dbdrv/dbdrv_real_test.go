package dbdrv

import (
	"os"
	"testing"
)

// TestRealDatabases testa os drivers nativos contra servidores reais iniciados via docker-compose.
// Ativado quando a variável LIAF_TEST_REAL_DB estiver definida como "1" ou "true".
func TestRealDatabases(t *testing.T) {
	if os.Getenv("LIAF_TEST_REAL_DB") != "1" && os.Getenv("LIAF_TEST_REAL_DB") != "true" {
		t.Skip("Pulando teste com bancos reais. Defina LIAF_TEST_REAL_DB=1 para executar.")
	}

	t.Run("PostgreSQL", func(t *testing.T) {
		dsn := os.Getenv("POSTGRES_DSN")
		if dsn == "" {
			dsn = "postgres://liaf:liaf_pass@localhost:5432/liaf_test?sslmode=disable"
		}
		conn, err := Open(DriverPostgres, dsn)
		if err != nil {
			t.Fatalf("Falha conectando ao PostgreSQL: %v", err)
		}
		defer conn.Close()

		if err := conn.Ping(); err != nil {
			t.Fatalf("Ping falhou: %v", err)
		}

		// Cria tabela
		_, err = conn.Exec("CREATE TABLE IF NOT EXISTS test_users (id INT PRIMARY KEY, name TEXT, email TEXT)", nil)
		if err != nil {
			t.Fatalf("CREATE TABLE falhou: %v", err)
		}

		// Limpa tabela
		_, err = conn.Exec("DELETE FROM test_users", nil)
		if err != nil {
			t.Fatalf("DELETE falhou: %v", err)
		}

		// Inserção com parâmetros
		affected, err := conn.Exec("INSERT INTO test_users (id, name, email) VALUES ($1, $2, $3)", []Value{1, "Alice", "alice@example.com"})
		if err != nil || affected != 1 {
			t.Fatalf("INSERT falhou: %v (affected=%d)", err, affected)
		}

		// Consulta com parâmetros
		rows, err := conn.Query("SELECT id, name, email FROM test_users WHERE id = $1", []Value{1})
		if err != nil {
			t.Fatalf("SELECT falhou: %v", err)
		}
		if len(rows.Values) != 1 {
			t.Fatalf("esperava 1 linha, obteve %d", len(rows.Values))
		}
		if rows.Values[0][1] != "Alice" {
			t.Fatalf("esperava nome Alice, obteve %v", rows.Values[0][1])
		}

		// Transação com Rollback
		if err := conn.Begin(); err != nil {
			t.Fatalf("BEGIN falhou: %v", err)
		}
		_, err = conn.Exec("INSERT INTO test_users (id, name, email) VALUES ($1, $2, $3)", []Value{2, "Bob", "bob@example.com"})
		if err != nil {
			t.Fatalf("INSERT na transação falhou: %v", err)
		}
		if err := conn.Rollback(); err != nil {
			t.Fatalf("ROLLBACK falhou: %v", err)
		}

		rows, err = conn.Query("SELECT id FROM test_users WHERE id = $1", []Value{2})
		if err != nil || len(rows.Values) != 0 {
			t.Fatalf("esperava que id 2 não existisse após rollback: rows=%v err=%v", rows, err)
		}
	})

	t.Run("MySQL", func(t *testing.T) {
		dsn := os.Getenv("MYSQL_DSN")
		if dsn == "" {
			dsn = "mysql://liaf:liaf_pass@localhost:3306/liaf_test?tls=false"
		}
		conn, err := Open(DriverMySQL, dsn)
		if err != nil {
			t.Fatalf("Falha conectando ao MySQL (caching_sha2_password): %v", err)
		}
		defer conn.Close()

		if err := conn.Ping(); err != nil {
			t.Fatalf("Ping falhou: %v", err)
		}

		_, err = conn.Exec("CREATE TABLE IF NOT EXISTS test_items (id INT PRIMARY KEY, title VARCHAR(100), price DOUBLE)", nil)
		if err != nil {
			t.Fatalf("CREATE TABLE falhou: %v", err)
		}

		_, err = conn.Exec("DELETE FROM test_items", nil)
		if err != nil {
			t.Fatalf("DELETE falhou: %v", err)
		}

		affected, err := conn.Exec("INSERT INTO test_items (id, title, price) VALUES (?, ?, ?)", []Value{10, "Produto X", 49.90})
		if err != nil || affected != 1 {
			t.Fatalf("INSERT falhou: %v (affected=%d)", err, affected)
		}

		rows, err := conn.Query("SELECT id, title, price FROM test_items WHERE id = ?", []Value{10})
		if err != nil {
			t.Fatalf("SELECT falhou: %v", err)
		}
		if len(rows.Values) != 1 {
			t.Fatalf("esperava 1 linha, obteve %d", len(rows.Values))
		}
		if rows.Values[0][1] != "Produto X" {
			t.Fatalf("esperava Produto X, obteve %v", rows.Values[0][1])
		}
	})

	t.Run("Redis", func(t *testing.T) {
		dsn := os.Getenv("REDIS_DSN")
		if dsn == "" {
			dsn = "redis://:liaf_redis_pass@localhost:6379/0"
		}
		conn, err := Open(DriverRedis, dsn)
		if err != nil {
			t.Fatalf("Falha conectando ao Redis: %v", err)
		}
		defer conn.Close()

		kv, ok := conn.(KV)
		if !ok {
			t.Fatalf("esperava que conexão Redis implementasse KV")
		}

		if err := kv.Set("test:key", "valor_secreto", 60); err != nil {
			t.Fatalf("SET falhou: %v", err)
		}

		val, found, err := kv.Get("test:key")
		if err != nil || !found || val != "valor_secreto" {
			t.Fatalf("GET falhou: val=%q found=%v err=%v", val, found, err)
		}

		_, found, err = kv.Get("test:nao_existe")
		if err != nil || found {
			t.Fatalf("chave inexistente deveria retornar found=false, obteve found=%v err=%v", found, err)
		}
	})
}
