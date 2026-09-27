package dbdrv

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	conn, err := Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)", nil); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	affected, err := conn.Exec("INSERT INTO users (name, age) VALUES (?, ?)", []Value{"Gabriel", int64(25)})
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if affected != 1 {
		t.Fatalf("Expected 1 row affected, got %d", affected)
	}

	rows, err := conn.Query("SELECT id, name, age FROM users WHERE name = ?", []Value{"Gabriel"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(rows.Values) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(rows.Values))
	}
	if rows.Values[0][1] != "Gabriel" {
		t.Fatalf("Expected 'Gabriel', got %v", rows.Values[0][1])
	}

	// Test transaction
	if err := conn.Begin(); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO users (name, age) VALUES (?, ?)", []Value{"Alice", int64(30)}); err != nil {
		t.Fatalf("Insert in tx failed: %v", err)
	}
	if err := conn.Rollback(); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	rowsAfterRollback, err := conn.Query("SELECT COUNT(*) FROM users", nil)
	if err != nil {
		t.Fatalf("Query count failed: %v", err)
	}
	if rowsAfterRollback.Values[0][0] != int64(1) {
		t.Fatalf("Expected count 1 after rollback, got %v", rowsAfterRollback.Values[0][0])
	}
}

// As PRAGMAs valem por conexao. Segurar duas conexoes ao mesmo tempo obriga o
// database/sql a abrir uma segunda, que tambem precisa sair configurada.
func TestSQLitePragmasApplyToEveryConnection(t *testing.T) {
	conn, err := Open("sqlite", filepath.Join(t.TempDir(), "pragmas.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	db := conn.(*sqliteConn).db

	ctx := context.Background()
	for i := 0; i < 2; i++ {
		c, err := db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		var timeout int
		var mode string
		if err := c.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&timeout); err != nil {
			t.Fatal(err)
		}
		if err := c.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
			t.Fatal(err)
		}
		if timeout != 5000 || mode != "wal" {
			t.Errorf("conexao %d: busy_timeout=%d journal_mode=%s", i+1, timeout, mode)
		}
	}
}

func TestSQLiteDefaultPragmasRespectDSN(t *testing.T) {
	suffix := "&_pragma=mmap_size(30000000000)&_pragma=cache_size(-64000)&_pragma=temp_store(MEMORY)"
	for _, tc := range []struct{ in, want string }{
		{"app.db", "app.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)" + suffix},
		{"app.db?pool_max=4", "app.db?pool_max=4&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)" + suffix},
		{"app.db?_pragma=journal_mode(DELETE)", "app.db?_pragma=journal_mode(DELETE)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)" + suffix},
	} {
		if got := withDefaultPragmas(tc.in); got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.in, got, tc.want)
		}
	}
}
