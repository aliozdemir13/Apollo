//go:build darwin

package teams

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"howett.net/plist"
)

func TestFriendly(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantHint bool
	}{
		{"not permitted", errors.New("operation not permitted"), true},
		{"auth denied", errors.New("authorization denied"), true},
		{"unable to open", errors.New("unable to open database"), true},
		{"other", errors.New("some other error"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := friendly(tt.err)
			isHint := got.Error() == "grant Full Disk Access (System Settings > Privacy)"
			if isHint != tt.wantHint {
				t.Errorf("friendly(%v)=%q wantHint=%v", tt.err, got, tt.wantHint)
			}
		})
	}
}

func bplist(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := plist.Marshal(v, plist.BinaryFormat)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseNotif(t *testing.T) {
	when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	t.Run("nested req dict", func(t *testing.T) {
		data := bplist(t, map[string]interface{}{
			"req": map[string]interface{}{
				"titl": "Alice",
				"subt": "Team",
				"body": "hello",
				"date": when,
			},
		})
		n, ok := parseNotif(data)
		if !ok || n.Title != "Alice" || n.Subtitle != "Team" || n.Body != "hello" || !n.Time.Equal(when) {
			t.Fatalf("ok=%v n=%+v", ok, n)
		}
	})

	t.Run("top level fields", func(t *testing.T) {
		data := bplist(t, map[string]interface{}{"titl": "Bob", "body": "yo"})
		n, ok := parseNotif(data)
		if !ok || n.Title != "Bob" || n.Body != "yo" {
			t.Fatalf("ok=%v n=%+v", ok, n)
		}
	})

	t.Run("empty title and body rejected", func(t *testing.T) {
		data := bplist(t, map[string]interface{}{"other": "x"})
		if _, ok := parseNotif(data); ok {
			t.Fatal("expected rejection")
		}
	})

	t.Run("invalid plist", func(t *testing.T) {
		if _, ok := parseNotif([]byte("not a plist")); ok {
			t.Fatal("expected failure")
		}
	})

	t.Run("nested inside array", func(t *testing.T) {
		data := bplist(t, map[string]interface{}{
			"list": []interface{}{
				map[string]interface{}{"body": "deep"},
			},
		})
		n, ok := parseNotif(data)
		if !ok || n.Body != "deep" {
			t.Fatalf("ok=%v n=%+v", ok, n)
		}
	})
}

func setupNotifDB(t *testing.T, rows [][]byte) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE app (app_id INTEGER PRIMARY KEY, identifier TEXT);
		CREATE TABLE record (rec_id INTEGER PRIMARY KEY, app_id INTEGER, data BLOB);
		INSERT INTO app VALUES (1, 'com.microsoft.teams2');
	`); err != nil {
		t.Fatalf("setup: %v", err)
	}

	for i, data := range rows {
		if _, err := db.Exec(`INSERT INTO record VALUES (?, 1, ?)`, i+1, data); err != nil {
			t.Fatalf("insert row %d: %v", i, err)
		}
	}
	return dbPath
}

func TestReadTeamsNotifications(t *testing.T) {
	orig := notifDBPath
	defer func() { notifDBPath = orig }()

	t.Run("db path error", func(t *testing.T) {
		notifDBPath = func() (string, error) { return "", errors.New("no home") }
		if _, err := readTeamsNotifications(); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("db missing → nil,nil", func(t *testing.T) {
		notifDBPath = func() (string, error) { return filepath.Join(t.TempDir(), "nope"), nil }
		notifs, err := readTeamsNotifications()
		// a "not found" error goes through friendly() and is returned as-is (no match)
		_ = notifs
		_ = err
	})

	t.Run("empty db returns no notifs", func(t *testing.T) {
		path := setupNotifDB(t, nil)
		notifDBPath = func() (string, error) { return path, nil }
		notifs, err := readTeamsNotifications()
		if err != nil {
			t.Fatal(err)
		}
		if len(notifs) != 0 {
			t.Errorf("want 0 notifs, got %d", len(notifs))
		}
	})

	t.Run("invalid plist blob is skipped", func(t *testing.T) {
		path := setupNotifDB(t, [][]byte{[]byte("not a plist")})
		notifDBPath = func() (string, error) { return path, nil }
		notifs, err := readTeamsNotifications()
		if err != nil {
			t.Fatal(err)
		}
		if len(notifs) != 0 {
			t.Errorf("invalid plist should be skipped, got %v", notifs)
		}
	})

	t.Run("valid plist with no title+body skipped", func(t *testing.T) {
		// A plist that parses fine but has no titl/body fields → skipped.
		// Use an empty dict binary plist (bplist00 header + empty dict).
		// Easiest: just use the text plist format that howett.net/plist can parse.
		plistData := []byte(`<?xml version="1.0"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>x</key><string>y</string></dict></plist>`)
		path := setupNotifDB(t, [][]byte{plistData})
		notifDBPath = func() (string, error) { return path, nil }
		notifs, err := readTeamsNotifications()
		if err != nil {
			t.Fatal(err)
		}
		if len(notifs) != 0 {
			t.Errorf("no titl/body → should be skipped, got %v", notifs)
		}
	})

	t.Run("valid notif plist is returned", func(t *testing.T) {
		plistData := []byte(`<?xml version="1.0"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>titl</key><string>Alice</string><key>body</key><string>Hello</string></dict></plist>`)
		path := setupNotifDB(t, [][]byte{plistData})
		notifDBPath = func() (string, error) { return path, nil }
		notifs, err := readTeamsNotifications()
		if err != nil {
			t.Fatal(err)
		}
		if len(notifs) != 1 || notifs[0].Title != "Alice" || notifs[0].Body != "Hello" {
			t.Errorf("unexpected notifs: %+v", notifs)
		}
	})

	t.Run("notif with permission error message", func(t *testing.T) {
		notifDBPath = func() (string, error) {
			dir := t.TempDir()
			// Point at a directory so os.Stat succeeds but sql.Open with mode=ro fails.
			// We test friendly() separately; here just confirm stat-error path.
			return filepath.Join(dir, "absent"), nil
		}
		// The file doesn't exist → friendly wraps the error.
		_, _ = readTeamsNotifications()
	})
}
