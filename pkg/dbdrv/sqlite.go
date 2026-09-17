package dbdrv

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type sqliteConn struct {
	db *sql.DB
	tx *sql.Tx
}

func openSQLite(dsn string) (Conn, error) {
	path := strings.TrimSpace(dsn)
	if strings.HasPrefix(path, "sqlite://") {
		path = strings.TrimPrefix(path, "sqlite://")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite open failed: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite ping failed: %w", err)
	}
	return &sqliteConn{db: db}, nil
}

func (c *sqliteConn) Exec(query string, args []Value) (int64, error) {
	var res sql.Result
	var err error
	if c.tx != nil {
		res, err = c.tx.Exec(query, args...)
	} else {
		res, err = c.db.Exec(query, args...)
	}
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (c *sqliteConn) Query(query string, args []Value) (*Rows, error) {
	var qRows *sql.Rows
	var err error
	if c.tx != nil {
		qRows, err = c.tx.Query(query, args...)
	} else {
		qRows, err = c.db.Query(query, args...)
	}
	if err != nil {
		return nil, err
	}
	defer qRows.Close()

	cols, err := qRows.Columns()
	if err != nil {
		return nil, err
	}
	rows := &Rows{Columns: cols}

	for qRows.Next() {
		colVals := make([]any, len(cols))
		colPointers := make([]any, len(cols))
		for i := range colVals {
			colPointers[i] = &colVals[i]
		}
		if err := qRows.Scan(colPointers...); err != nil {
			return nil, err
		}
		rowValues := make([]Value, len(cols))
		for i, val := range colVals {
			switch v := val.(type) {
			case nil:
				rowValues[i] = nil
			case bool:
				rowValues[i] = v
			case int64:
				rowValues[i] = v
			case int:
				rowValues[i] = int64(v)
			case float64:
				rowValues[i] = v
			case []byte:
				rowValues[i] = string(v)
			case string:
				rowValues[i] = v
			default:
				rowValues[i] = fmt.Sprint(v)
			}
		}
		rows.Values = append(rows.Values, rowValues)
	}
	if err := qRows.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

func (c *sqliteConn) Begin() error {
	if c.tx != nil {
		return fmt.Errorf("transaction already in progress")
	}
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	c.tx = tx
	return nil
}

func (c *sqliteConn) Commit() error {
	if c.tx == nil {
		return fmt.Errorf("no transaction in progress")
	}
	err := c.tx.Commit()
	c.tx = nil
	return err
}

func (c *sqliteConn) Rollback() error {
	if c.tx == nil {
		return fmt.Errorf("no transaction in progress")
	}
	err := c.tx.Rollback()
	c.tx = nil
	return err
}

func (c *sqliteConn) Ping() error {
	if c.db == nil {
		return ErrConnLost
	}
	return c.db.Ping()
}

func (c *sqliteConn) Close() error {
	if c.tx != nil {
		_ = c.tx.Rollback()
		c.tx = nil
	}
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}
