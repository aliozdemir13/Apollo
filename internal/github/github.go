// Package github fetches open pull requests from a set of selected repos using
// the GitHub REST API and a personal access token.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Service talks to the GitHub API.
type Service struct {
	client *http.Client
}

// New returns a Service.
func New() *Service {
	return &Service{client: &http.Client{Timeout: 15 * time.Second}}
}

// apiBase is the GitHub API root, overridable in tests.
var apiBase = "https://api.github.com"

// GetPRs returns open PRs across the given repos. If login is non-empty, only
// PRs authored by that user are kept.
func (s *Service) GetPRs(ctx context.Context, token string, repos []string, login string) (Result, error) {
	var res Result
	if strings.TrimSpace(token) == "" {
		return res, fmt.Errorf("no GitHub token configured")
	}

	var mu sync.Mutex
	g, ctx := errgroup.WithContext(ctx)

	for _, repo := range repos {
		repo := strings.TrimSpace(repo) // Create local copy for closure safety
		if repo == "" {
			continue
		}

		g.Go(func() error {
			prs, err := s.repoPRs(ctx, token, repo)
			if err != nil {
				mu.Lock()
				res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", repo, err))
				mu.Unlock()
				return nil // Return nil so other repo fetches aren't cancelled
			}

			var localPRs []PR
			for _, p := range prs {
				if login != "" && !strings.EqualFold(p.User.Login, login) {
					continue
				}
				localPRs = append(localPRs, PR{
					Repo:      repo,
					Number:    p.Number,
					Title:     p.Title,
					Author:    p.User.Login,
					URL:       p.HTMLURL,
					Draft:     p.Draft,
					UpdatedAt: p.UpdatedAt,
				})
			}

			mu.Lock()
			res.PRs = append(res.PRs, localPRs...)
			mu.Unlock()
			return nil
		})
	}

	_ = g.Wait() // All fetches executed in parallel!

	// Most recently updated first
	sort.Slice(res.PRs, func(i, j int) bool {
		return res.PRs[i].UpdatedAt > res.PRs[j].UpdatedAt
	})
	return res, nil
}

func (s *Service) repoPRs(ctx context.Context, token, repo string) ([]ApiPR, error) {
	q := url.Values{}
	q.Set("state", "open")
	q.Set("per_page", "50")
	q.Set("sort", "updated")
	q.Set("direction", "desc")
	u := fmt.Sprintf("%s/repos/%s/pulls?", apiBase, repo) + q.Encode()

	var prs []ApiPR
	if err := s.apiGet(ctx, token, u, &prs); err != nil {
		slog.Error("Failed to fetch repo PRs", "repo", repo, "error", err)
		return nil, err
	}
	return prs, nil
}

// apiGet performs an authenticated GET and decodes the JSON body into out.
func (s *Service) apiGet(ctx context.Context, token, u string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Apollo-Widget/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		slog.Error("Repo not found or no access", "url", u, "statusCode", resp.StatusCode, "response", &resp)
		return fmt.Errorf("unauthorized (check token)")
	case http.StatusNotFound:
		slog.Error("Repo not found or no access", "url", u, "statusCode", resp.StatusCode, "response", &resp)
		return fmt.Errorf("not found or no access")
	default:
		slog.Error("Unexpected response", "url", u, "statusCode", resp.StatusCode, "response", &resp)
		return fmt.Errorf("status %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// GetReviewRequests returns open PRs where the authenticated user is requested
// as a reviewer, scoped to the configured repos (or all if none configured).
func (s *Service) GetReviewRequests(ctx context.Context, token string, repos []string) (Result, error) {
	var res Result
	if strings.TrimSpace(token) == "" {
		return res, fmt.Errorf("no GitHub token configured")
	}

	q := "is:open is:pr review-requested:@me"
	for _, repo := range repos {
		if repo = strings.TrimSpace(repo); repo != "" {
			q += " repo:" + repo
		}
	}
	u := apiBase + "/search/issues?per_page=50&q=" + url.QueryEscape(q)

	var out struct {
		Items []struct {
			Number        int    `json:"number"`
			Title         string `json:"title"`
			HTMLURL       string `json:"html_url"`
			UpdatedAt     string `json:"updated_at"`
			Draft         bool   `json:"draft"`
			RepositoryURL string `json:"repository_url"`
			User          struct {
				Login string `json:"login"`
			} `json:"user"`
		} `json:"items"`
	}
	if err := s.apiGet(ctx, token, u, &out); err != nil {
		return res, err
	}

	for _, it := range out.Items {
		res.PRs = append(res.PRs, PR{
			Repo:      repoFromAPIURL(it.RepositoryURL),
			Number:    it.Number,
			Title:     it.Title,
			Author:    it.User.Login,
			URL:       it.HTMLURL,
			Draft:     it.Draft,
			UpdatedAt: it.UpdatedAt,
		})
	}
	sort.Slice(res.PRs, func(i, j int) bool { return res.PRs[i].UpdatedAt > res.PRs[j].UpdatedAt })
	return res, nil
}

// repoFromAPIURL turns "https://api.github.com/repos/owner/name" into "owner/name".
func repoFromAPIURL(u string) string {
	const marker = "/repos/"
	if i := strings.Index(u, marker); i >= 0 {
		return u[i+len(marker):]
	}
	return u
}

// GetWorkflowRuns returns the most recent Actions run for each configured repo.
func (s *Service) GetWorkflowRuns(ctx context.Context, token string, repos []string) (WorkflowResult, error) {
	var res WorkflowResult
	if strings.TrimSpace(token) == "" {
		return res, fmt.Errorf("no GitHub token configured")
	}

	for _, repo := range repos {
		repo = strings.TrimSpace(repo)
		if repo == "" {
			continue
		}
		u := fmt.Sprintf("%s/repos/%s/actions/runs?per_page=1", apiBase, repo)
		var out struct {
			WorkflowRuns []struct {
				Name       string `json:"name"`
				HeadBranch string `json:"head_branch"`
				Status     string `json:"status"`
				Conclusion string `json:"conclusion"`
				Event      string `json:"event"`
				HTMLURL    string `json:"html_url"`
				UpdatedAt  string `json:"updated_at"`
			} `json:"workflow_runs"`
		}
		if err := s.apiGet(ctx, token, u, &out); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", repo, err))
			continue
		}
		if len(out.WorkflowRuns) == 0 {
			continue
		}
		r := out.WorkflowRuns[0]
		res.Runs = append(res.Runs, WorkflowRun{
			Repo:       repo,
			Name:       r.Name,
			Status:     r.Status,
			Conclusion: r.Conclusion,
			Branch:     r.HeadBranch,
			Event:      r.Event,
			URL:        r.HTMLURL,
			UpdatedAt:  r.UpdatedAt,
		})
	}
	sort.Slice(res.Runs, func(i, j int) bool { return res.Runs[i].UpdatedAt > res.Runs[j].UpdatedAt })
	return res, nil
}
