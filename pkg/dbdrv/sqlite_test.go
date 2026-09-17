package dbdrv

import (
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
