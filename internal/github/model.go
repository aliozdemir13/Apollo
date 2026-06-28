package github

// PR is a single open pull request, flattened for display.
type PR struct {
	Repo      string `json:"repo"` // "owner/name"
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	URL       string `json:"url"`
	Draft     bool   `json:"draft"`
	UpdatedAt string `json:"updatedAt"`
}

// Result wraps the PR list plus any per-repo errors so the UI can still show
// what succeeded.
type Result struct {
	PRs    []PR     `json:"prs"`
	Errors []string `json:"errors"`
}

type ApiPR struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	HTMLURL   string `json:"html_url"`
	Draft     bool   `json:"draft"`
	UpdatedAt string `json:"updated_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

// WorkflowRun is the latest GitHub Actions run for a repo.
type WorkflowRun struct {
	Repo       string `json:"repo"`
	Name       string `json:"name"`
	Status     string `json:"status"`     // queued | in_progress | completed
	Conclusion string `json:"conclusion"` // success | failure | cancelled | ""
	Branch     string `json:"branch"`
	Event      string `json:"event"`
	URL        string `json:"url"`
	UpdatedAt  string `json:"updatedAt"`
}

// WorkflowResult wraps the runs plus per-repo errors.
type WorkflowResult struct {
	Runs   []WorkflowRun `json:"runs"`
	Errors []string      `json:"errors"`
}

type ReviewRequestsResult struct {
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
