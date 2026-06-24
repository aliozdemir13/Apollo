package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRepoFromAPIURL(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"https://api.github.com/repos/owner/name", "owner/name"},
		{"https://api.github.com/repos/a/b/c", "a/b/c"},
		{"no-marker-here", "no-marker-here"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := repoFromAPIURL(tt.in); got != tt.want {
				t.Errorf("repoFromAPIURL(%q)=%q want %q", tt.in, got, tt.want)
			}
		})
	}
}

// router serves canned responses keyed by URL-path substring.
func router(routes map[string]struct {
	status int
	body   string
}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for sub, resp := range routes {
			if strings.Contains(r.URL.Path, sub) {
				w.WriteHeader(resp.status)
				_, _ = w.Write([]byte(resp.body))
				return
			}
		}
		w.WriteHeader(404)
	}))
}

func withBase(t *testing.T, url string) {
	t.Helper()
	old := apiBase
	apiBase = url
	t.Cleanup(func() { apiBase = old })
}

func TestGetPRs(t *testing.T) {
	t.Run("no token", func(t *testing.T) {
		if _, err := New().GetPRs(context.Background(), "  ", []string{"a/b"}, ""); err == nil {
			t.Fatal("want error for empty token")
		}
	})

	t.Run("ok with login filter and empty repo skipped", func(t *testing.T) {
		srv := router(map[string]struct {
			status int
			body   string
		}{
			"/repos/o/r1/pulls": {200, `[
				{"number":1,"title":"A","html_url":"u1","draft":false,"updated_at":"2026-01-02","user":{"login":"me"}},
				{"number":2,"title":"B","html_url":"u2","draft":true,"updated_at":"2026-01-03","user":{"login":"other"}}
			]`},
		})
		defer srv.Close()
		withBase(t, srv.URL)

		res, err := New().GetPRs(context.Background(), "tok", []string{"o/r1", "  "}, "me")
		if err != nil {
			t.Fatal(err)
		}
		if len(res.PRs) != 1 || res.PRs[0].Author != "me" || res.PRs[0].Repo != "o/r1" {
			t.Fatalf("got %+v", res.PRs)
		}
	})

	t.Run("sorted desc and no filter", func(t *testing.T) {
		srv := router(map[string]struct {
			status int
			body   string
		}{
			"/pulls": {200, `[
				{"number":1,"title":"old","updated_at":"2026-01-01","user":{"login":"x"}},
				{"number":2,"title":"new","updated_at":"2026-02-01","user":{"login":"y"}}
			]`},
		})
		defer srv.Close()
		withBase(t, srv.URL)
		res, _ := New().GetPRs(context.Background(), "tok", []string{"o/r"}, "")
		if len(res.PRs) != 2 || res.PRs[0].Title != "new" {
			t.Fatalf("not sorted: %+v", res.PRs)
		}
	})

	errStatusCases := []struct {
		name   string
		status int
	}{
		{"unauthorized", 401},
		{"not found", 404},
		{"server error", 500},
	}
	for _, tt := range errStatusCases {
		t.Run("repo error "+tt.name, func(t *testing.T) {
			srv := router(map[string]struct {
				status int
				body   string
			}{"/pulls": {tt.status, ``}})
			defer srv.Close()
			withBase(t, srv.URL)
			res, err := New().GetPRs(context.Background(), "tok", []string{"o/r"}, "")
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Errors) != 1 {
				t.Fatalf("want 1 repo error, got %v", res.Errors)
			}
		})
	}

	t.Run("bad json", func(t *testing.T) {
		srv := router(map[string]struct {
			status int
			body   string
		}{"/pulls": {200, `{bad`}})
		defer srv.Close()
		withBase(t, srv.URL)
		res, _ := New().GetPRs(context.Background(), "tok", []string{"o/r"}, "")
		if len(res.Errors) != 1 {
			t.Fatalf("want decode error recorded, got %v", res.Errors)
		}
	})

	t.Run("request build error", func(t *testing.T) {
		withBase(t, "http://\x7f")
		res, _ := New().GetPRs(context.Background(), "tok", []string{"o/r"}, "")
		if len(res.Errors) != 1 {
			t.Fatalf("want build error recorded, got %v", res.Errors)
		}
	})
}

func TestGetReviewRequests(t *testing.T) {
	t.Run("no token", func(t *testing.T) {
		if _, err := New().GetReviewRequests(context.Background(), "", nil); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("ok", func(t *testing.T) {
		srv := router(map[string]struct {
			status int
			body   string
		}{"/search/issues": {200, `{"items":[
			{"number":7,"title":"old","html_url":"u","updated_at":"2026-03-01","repository_url":"https://api.github.com/repos/o/r","user":{"login":"a"}},
			{"number":8,"title":"new","html_url":"u2","updated_at":"2026-04-01","repository_url":"https://api.github.com/repos/o/r2","user":{"login":"b"}}
		]}`}})
		defer srv.Close()
		withBase(t, srv.URL)
		res, err := New().GetReviewRequests(context.Background(), "tok", []string{"o/r", " "})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.PRs) != 2 || res.PRs[0].Number != 8 || res.PRs[1].Repo != "o/r" {
			t.Fatalf("got %+v", res.PRs)
		}
	})

	t.Run("http error", func(t *testing.T) {
		srv := router(map[string]struct {
			status int
			body   string
		}{"/search/issues": {500, ``}})
		defer srv.Close()
		withBase(t, srv.URL)
		if _, err := New().GetReviewRequests(context.Background(), "tok", nil); err == nil {
			t.Fatal("want error")
		}
	})
}

func TestGetWorkflowRuns(t *testing.T) {
	t.Run("no token", func(t *testing.T) {
		if _, err := New().GetWorkflowRuns(context.Background(), "", nil); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("ok with one empty repo and one error repo", func(t *testing.T) {
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			switch {
			case strings.Contains(r.URL.Path, "/repos/o/good/"):
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"workflow_runs":[{"name":"CI","head_branch":"main","status":"completed","conclusion":"success","event":"push","html_url":"u","updated_at":"2026-01-05"}]}`))
			case strings.Contains(r.URL.Path, "/repos/o/good2/"):
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"workflow_runs":[{"name":"CI2","head_branch":"dev","status":"completed","conclusion":"failure","event":"push","html_url":"u2","updated_at":"2026-02-09"}]}`))
			case strings.Contains(r.URL.Path, "/repos/o/empty/"):
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"workflow_runs":[]}`))
			default:
				w.WriteHeader(500)
			}
		}))
		defer srv.Close()
		withBase(t, srv.URL)
		res, err := New().GetWorkflowRuns(context.Background(), "tok", []string{"o/good", "o/good2", "o/empty", "o/bad", "  "})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Runs) != 2 || res.Runs[0].Repo != "o/good2" || res.Runs[1].Conclusion != "success" {
			t.Fatalf("runs=%+v", res.Runs)
		}
		if len(res.Errors) != 1 {
			t.Fatalf("errors=%v", res.Errors)
		}
	})
}

func TestApiGetBuildError(t *testing.T) {
	// invalid base URL → request build error path in apiGet.
	withBase(t, "http://\x7f")
	if _, err := New().GetReviewRequests(context.Background(), "tok", nil); err == nil {
		t.Fatal("want build error")
	}
}

// TestApiGetStatusBranches exercises apiGet's 401/404/decode branches via the
// reviews endpoint.
func TestApiGetStatusBranches(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", 401, ``},
		{"not found", 404, ``},
		{"bad json", 200, `{not-json`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			srv := router(map[string]struct {
				status int
				body   string
			}{"/search/issues": {tt.status, tt.body}})
			defer srv.Close()
			withBase(t, srv.URL)
			if _, err := New().GetReviewRequests(context.Background(), "tok", nil); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

func TestConnectionErrors(t *testing.T) {
	srv := router(map[string]struct {
		status int
		body   string
	}{"/": {200, `[]`}})
	url := srv.URL
	srv.Close() // force connection errors
	withBase(t, url)

	// apiGet transport error
	if _, err := New().GetReviewRequests(context.Background(), "tok", nil); err == nil {
		t.Fatal("want conn error from reviews")
	}
	// repoPRs transport error → recorded as a per-repo error
	res, _ := New().GetPRs(context.Background(), "tok", []string{"o/r"}, "")
	if len(res.Errors) != 1 {
		t.Fatalf("want repo error from conn failure, got %v", res.Errors)
	}
}
