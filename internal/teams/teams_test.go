package teams

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

type fakeMarshaler struct {
	data []byte
	err  error
}

func (f fakeMarshaler) Marshal() ([]byte, error) { return f.data, f.err }

type fakeUnmarshaler struct {
	got *[]byte
	err error
}

func (f fakeUnmarshaler) Unmarshal(b []byte) error {
	if f.got != nil {
		*f.got = b
	}
	return f.err
}

func TestFileCache(t *testing.T) {
	ctx := context.Background()

	t.Run("replace missing file is noop", func(t *testing.T) {
		fc := &fileCache{path: filepath.Join(t.TempDir(), "nope")}
		if err := fc.Replace(ctx, fakeUnmarshaler{}, cache.ReplaceHints{}); err != nil {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("export then replace round-trips", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "cache.bin")
		fc := &fileCache{path: path}
		if err := fc.Export(ctx, fakeMarshaler{data: []byte("token-blob")}, cache.ExportHints{}); err != nil {
			t.Fatal(err)
		}
		var got []byte
		if err := fc.Replace(ctx, fakeUnmarshaler{got: &got}, cache.ReplaceHints{}); err != nil {
			t.Fatal(err)
		}
		if string(got) != "token-blob" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("export marshal error", func(t *testing.T) {
		fc := &fileCache{path: filepath.Join(t.TempDir(), "x")}
		if err := fc.Export(ctx, fakeMarshaler{err: errors.New("boom")}, cache.ExportHints{}); err == nil {
			t.Fatal("want marshal error")
		}
	})

	t.Run("replace read error (path is dir)", func(t *testing.T) {
		dir := t.TempDir() // a directory, ReadFile fails with non-NotExist error
		fc := &fileCache{path: dir}
		if err := fc.Replace(ctx, fakeUnmarshaler{}, cache.ReplaceHints{}); err == nil {
			t.Fatal("want read error")
		}
	})

	t.Run("replace unmarshal error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "c.bin")
		_ = os.WriteFile(path, []byte("data"), 0o600)
		fc := &fileCache{path: path}
		if err := fc.Replace(ctx, fakeUnmarshaler{err: errors.New("bad")}, cache.ReplaceHints{}); err == nil {
			t.Fatal("want unmarshal error")
		}
	})
}

func TestMatchesAny(t *testing.T) {
	tests := []struct {
		name string
		in   string
		subs []string
		want bool
	}{
		{"match ci", "Project ACME", []string{"acme"}, true},
		{"no match", "Project ACME", []string{"zzz"}, false},
		{"empty sub skipped", "Project", []string{"  ", "pro"}, true},
		{"all empty", "Project", []string{"", "   "}, false},
		{"no subs", "Project", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesAny(tt.in, tt.subs); got != tt.want {
				t.Errorf("matchesAny(%q,%v)=%v want %v", tt.in, tt.subs, got, tt.want)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		isZero bool
	}{
		{"empty", "", true},
		{"rfc3339", "2026-01-02T15:04:05Z", false},
		{"fractional", "2026-01-02T15:04:05.1234567Z", false},
		{"invalid", "not-a-time", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTime(tt.in)
			if got.IsZero() != tt.isZero {
				t.Errorf("parseTime(%q).IsZero()=%v want %v", tt.in, got.IsZero(), tt.isZero)
			}
		})
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"<b>hi</b>", "hi"},
		{"plain", "plain"},
		{"<p>a<br>b</p>", "ab"},
		{"unbalanced >text", "unbalanced text"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := stripHTML(tt.in); got != tt.want {
				t.Errorf("stripHTML(%q)=%q want %q", tt.in, got, tt.want)
			}
		})
	}
}

func mkChat(topic string, members []string, read, last, body, ctype, from string) GraphChat {
	var c GraphChat
	c.Topic = topic
	c.Viewpoint.LastMessageReadDateTime = read
	c.LastMessagePreview.CreatedDateTime = last
	c.LastMessagePreview.Body.Content = body
	c.LastMessagePreview.Body.ContentType = ctype
	c.LastMessagePreview.From.User.DisplayName = from
	for _, m := range members {
		c.Members = append(c.Members, struct {
			DisplayName string `json:"displayName"`
		}{DisplayName: m})
	}
	return c
}

func TestGraphChat(t *testing.T) {
	long := ""
	for i := 0; i < 100; i++ {
		long += "x"
	}

	t.Run("unread", func(t *testing.T) {
		tests := []struct {
			name string
			read string
			last string
			want bool
		}{
			{"newer than read", "2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z", true},
			{"older than read", "2026-01-03T00:00:00Z", "2026-01-02T00:00:00Z", false},
			{"no last message", "2026-01-01T00:00:00Z", "", false},
			{"never read", "", "2026-01-02T00:00:00Z", true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := mkChat("", nil, tt.read, tt.last, "b", "text", "f")
				if got := c.unread(); got != tt.want {
					t.Errorf("unread=%v want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("displayName", func(t *testing.T) {
		tests := []struct {
			name    string
			topic   string
			members []string
			want    string
		}{
			{"topic wins", "Team Chat", []string{"A", "B"}, "Team Chat"},
			{"members joined", "", []string{"Alice", "Bob"}, "Alice, Bob"},
			{"members with blanks", "  ", []string{" ", "Bob"}, "Bob"},
			{"fallback", "", nil, "(chat)"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := mkChat(tt.topic, tt.members, "", "", "", "", "")
				if got := c.displayName(); got != tt.want {
					t.Errorf("displayName=%q want %q", got, tt.want)
				}
			})
		}
	})

	t.Run("sender", func(t *testing.T) {
		c := mkChat("", nil, "", "", "", "", "Carol")
		if c.sender() != "Carol" {
			t.Errorf("sender=%q", c.sender())
		}
	})

	t.Run("preview", func(t *testing.T) {
		tests := []struct {
			name  string
			body  string
			ctype string
			want  string
		}{
			{"html stripped", "<b>hi</b>", "html", "hi"},
			{"plain", "hello", "text", "hello"},
			{"truncated", long, "text", long[:80] + "…"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := mkChat("", nil, "", "", tt.body, tt.ctype, "")
				if got := c.preview(); got != tt.want {
					t.Errorf("preview=%q want %q", got, tt.want)
				}
			})
		}
	})
}

func TestNewDefaultsTenant(t *testing.T) {
	s := New("client", "", "/tmp/none")
	if s.tenantID != "common" {
		t.Errorf("tenant=%q want common", s.tenantID)
	}
	if !s.Configured() {
		t.Error("should be configured with client id")
	}
	if New("", "", "/tmp/none").Configured() {
		t.Error("empty client id should be unconfigured")
	}
}

func TestGetUnreadNeedsLogin(t *testing.T) {
	// No client id → cannot get a token → NeedsLogin, no error.
	res, err := New("", "", t.TempDir()+"/cache").GetUnread(context.Background(), nil)
	if err != nil || !res.NeedsLogin {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	// With a client id, MSAL client builds and finds no cached account → still NeedsLogin.
	res2, err2 := New("test-client-id", "common", t.TempDir()+"/cache").GetUnread(context.Background(), nil)
	if err2 != nil || !res2.NeedsLogin {
		t.Fatalf("res2=%+v err2=%v", res2, err2)
	}
	if New("", "", t.TempDir()+"/c").LoggedIn(context.Background()) {
		t.Error("LoggedIn should be false without token")
	}
}

func TestFetchChats(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		badBase string
		wantErr bool
		wantLen int
	}{
		{name: "ok", status: 200, body: `{"value":[{"topic":"T"},{"topic":"U"}]}`, wantLen: 2},
		{name: "http error", status: 500, body: ``, wantErr: true},
		{name: "bad json", status: 200, body: `{bad`, wantErr: true},
		{name: "bad base", badBase: "http://\x7f", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := graphBase
			defer func() { graphBase = old }()
			if tt.badBase != "" {
				graphBase = tt.badBase
			} else {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.status)
					_, _ = w.Write([]byte(tt.body))
				}))
				defer srv.Close()
				graphBase = srv.URL
			}
			chats, err := fetchChats(context.Background(), "tok")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if !tt.wantErr && len(chats) != tt.wantLen {
				t.Errorf("len=%d want %d", len(chats), tt.wantLen)
			}
		})
	}
}

func TestFetchChatsConnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	old := graphBase
	defer func() { graphBase = old }()
	graphBase = url
	if _, err := fetchChats(context.Background(), "tok"); err == nil {
		t.Fatal("want conn error")
	}
}

func TestGetUnreadLocal(t *testing.T) {
	old := readNotifs
	defer func() { readNotifs = old }()

	t.Run("groups, filters, dedups", func(t *testing.T) {
		readNotifs = func() ([]Notif, error) {
			return []Notif{
				{Title: "Alice", Subtitle: "Team", Body: "hi", Time: time.Unix(100, 0)},
				{Title: "Alice", Body: "older", Time: time.Unix(50, 0)}, // dup chat, dropped
				{Title: "", Body: "no title", Time: time.Time{}},        // empty title -> "Teams"
				{Title: "Bob", Body: "yo", Time: time.Unix(80, 0)},
			}, nil
		}
		res, err := (&Service{}).GetUnreadLocal(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.TotalUnread != 3 {
			t.Fatalf("total=%d want 3 (%+v)", res.TotalUnread, res.UnreadChats)
		}
		if res.UnreadChats[0].Name != "Alice" || res.UnreadChats[0].Timestamp == "" {
			t.Errorf("first=%+v", res.UnreadChats[0])
		}
	})

	t.Run("name filter", func(t *testing.T) {
		readNotifs = func() ([]Notif, error) {
			return []Notif{{Title: "Alice", Body: "x"}, {Title: "Bob", Body: "y"}}, nil
		}
		res, _ := (&Service{}).GetUnreadLocal(context.Background(), []string{"bob"})
		if len(res.UnreadChats) != 1 || res.UnreadChats[0].Name != "Bob" {
			t.Fatalf("got %+v", res.UnreadChats)
		}
	})

	t.Run("reader error", func(t *testing.T) {
		readNotifs = func() ([]Notif, error) { return nil, errors.New("boom") }
		if _, err := (&Service{}).GetUnreadLocal(context.Background(), nil); err == nil {
			t.Fatal("want error")
		}
	})
}
