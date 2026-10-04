// Package database uses the existing Rails SQLite schema and explicit SQL.
package database

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schema string

var migrations = []string{"20231215043540", "20231220143106", "20240110071740", "20240115124901", "20240130003150", "20240130213001", "20240131105830", "20240209110503", "20250825100957", "20250825100958", "20250825100959", "20251126092013", "20251126115722", "20251126130131", "20251212154340"}

type accountCacheEntry struct {
	gen     uint64
	account *Account
}

// A single writer prevents pool starvation while WAL readers proceed independently.
type DB struct {
	ResetConnections    func(int64)
	PurgeBlobs          func([]int64)
	RemoveBannedContent func(int64)
	AfterAccountQuery   func()
	Read                *readPool
	Write               *sql.DB
	Now                 func() time.Time
	accountCache        atomic.Pointer[accountCacheEntry]
}

func Open(path string, readers int) (*DB, error) {
	if readers < 1 {
		return nil, errors.New("database readers must be positive")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	uri := (&url.URL{Scheme: "file", Path: path}).String()
	options := "?_busy_timeout=5000&_foreign_keys=on&_journal_mode=WAL&_synchronous=NORMAL&_cache_size=2000"
	// Reuse transaction statements on the single writer connection. The driver
	// resets bindings on reuse; results and authorization are never cached.
	w, err := sql.Open("sqlite3", uri+options+"&_txlock=immediate&_stmt_cache_size=64")
	if err != nil {
		return nil, err
	}
	w.SetMaxOpenConns(1)
	w.SetMaxIdleConns(1)
	fail := func(err error) (*DB, error) { w.Close(); return nil, err }
	if err = w.Ping(); err != nil {
		return fail(err)
	}
	if err = prepare(w); err != nil {
		return fail(err)
	}
	r, err := sql.Open("sqlite3", uri+options+"&mode=ro&_query_only=on")
	if err != nil {
		return fail(err)
	}
	r.SetMaxOpenConns(readers)
	r.SetMaxIdleConns(readers)
	if err = r.Ping(); err != nil {
		r.Close()
		return fail(err)
	}
	now := time.Now
	if raw := os.Getenv("CAMPFIRE_FROZEN_TIME"); raw != "" {
		frozen, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			r.Close()
			return fail(err)
		}
		now = func() time.Time { return frozen }
	}
	return &DB{Read: &readPool{DB: r, statements: make(map[string]*sql.Stmt)}, Write: w, Now: now}, nil
}
func (d *DB) Close() error     { return errors.Join(d.Read.Close(), d.Write.Close()) }
func Stamp(t time.Time) string { return t.UTC().Format("2006-01-02 15:04:05.000000") }
func (d *DB) Transaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := d.Write.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
func prepare(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists int
	if err = tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		var count int
		if err = tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return errors.New("refusing to initialize a nonempty database without schema_migrations")
		}
		if _, err = tx.Exec(schema); err != nil {
			return err
		}
		for i := len(migrations) - 1; i >= 0; i-- {
			if _, err = tx.Exec("INSERT INTO schema_migrations(version) VALUES (?)", migrations[i]); err != nil {
				return err
			}
		}
		now := Stamp(time.Now())
		for _, p := range [][2]string{{"environment", "production"}, {"schema_sha1", "f75da8dad38bfb179ffd757bd7a7c2b3f818bc29"}} {
			if _, err = tx.Exec("INSERT INTO ar_internal_metadata(key,value,created_at,updated_at) VALUES (?,?,?,?)", p[0], p[1], now, now); err != nil {
				return err
			}
		}
	}
	for _, v := range migrations {
		var found int
		if err = tx.QueryRow("SELECT count(*) FROM schema_migrations WHERE version=?", v).Scan(&found); err != nil {
			return err
		}
		if found != 1 {
			return fmt.Errorf("pending migration %s: migrate with the reference app before starting", v)
		}
	}
	if _, err = tx.Exec("CREATE INDEX IF NOT EXISTS index_messages_on_room_id_and_created_at ON messages(room_id,created_at)"); err != nil {
		return err
	}
	return tx.Commit()
}

var timestampLayouts = []string{"2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05.999999999-07:00", time.RFC3339Nano}

func parseSQLiteTimestamp(raw string) (time.Time, bool) {
	if len(raw) < 19 || raw[4] != '-' || raw[7] != '-' || (raw[10] != ' ' && raw[10] != 'T') || raw[13] != ':' || raw[16] != ':' {
		return time.Time{}, false
	}
	y0, y1, y2, y3 := raw[0]-'0', raw[1]-'0', raw[2]-'0', raw[3]-'0'
	m0, m1 := raw[5]-'0', raw[6]-'0'
	d0, d1 := raw[8]-'0', raw[9]-'0'
	h0, h1 := raw[11]-'0', raw[12]-'0'
	min0, min1 := raw[14]-'0', raw[15]-'0'
	s0, s1 := raw[17]-'0', raw[18]-'0'
	if y0 > 9 || y1 > 9 || y2 > 9 || y3 > 9 ||
		m0 > 9 || m1 > 9 || d0 > 9 || d1 > 9 ||
		h0 > 9 || h1 > 9 || min0 > 9 || min1 > 9 ||
		s0 > 9 || s1 > 9 {
		return time.Time{}, false
	}
	year := int(y0)*1000 + int(y1)*100 + int(y2)*10 + int(y3)
	month := int(m0)*10 + int(m1)
	day := int(d0)*10 + int(d1)
	hour := int(h0)*10 + int(h1)
	min := int(min0)*10 + int(min1)
	sec := int(s0)*10 + int(s1)
	if month < 1 || month > 12 || day < 1 || day > 31 || hour > 23 || min > 59 || sec > 59 {
		return time.Time{}, false
	}
	nsec := 0
	rem := raw[19:]
	if len(rem) > 0 && rem[0] == '.' {
		rem = rem[1:]
		mul := 100000000
		i := 0
		for i < len(rem) && rem[i] >= '0' && rem[i] <= '9' {
			if mul > 0 {
				nsec += int(rem[i]-'0') * mul
				mul /= 10
			}
			i++
		}
		rem = rem[i:]
	}
	if len(rem) > 0 && rem != "Z" && rem != "+00:00" && rem != "-00:00" {
		return time.Time{}, false
	}
	return time.Date(year, time.Month(month), day, hour, min, sec, nsec, time.UTC), true
}

// SQLite's DATETIME(6) declaration is returned as text by go-sqlite3.
type timestamp struct{ value *time.Time }

func (t timestamp) Scan(value any) error {
	if v, ok := value.(time.Time); ok {
		*t.value = v
		return nil
	}
	var raw string
	switch v := value.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	default:
		return fmt.Errorf("invalid timestamp type %T", value)
	}
	if parsed, ok := parseSQLiteTimestamp(raw); ok {
		*t.value = parsed
		return nil
	}
	for _, layout := range timestampLayouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			*t.value = parsed
			return nil
		}
	}
	return fmt.Errorf("invalid timestamp %q", raw)
}
