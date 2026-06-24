//go:build darwin

package teams

import (
	"errors"
	"testing"
	"time"

	"howett.net/plist"
)

func TestFriendly(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantHint  bool
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
