package teams

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// graphBase is the Microsoft Graph API root, overridable in tests.
var graphBase = "https://graph.microsoft.com/v1.0"

func (c GraphChat) unread() bool {
	last := parseTime(c.LastMessagePreview.CreatedDateTime)
	if last.IsZero() {
		return false
	}
	read := parseTime(c.Viewpoint.LastMessageReadDateTime)
	return last.After(read)
}

func (c GraphChat) displayName() string {
	if strings.TrimSpace(c.Topic) != "" {
		return c.Topic
	}
	// For 1:1 / group chats with no topic, join member names.
	var names []string
	for _, m := range c.Members {
		if n := strings.TrimSpace(m.DisplayName); n != "" {
			names = append(names, n)
		}
	}
	if len(names) > 0 {
		return strings.Join(names, ", ")
	}
	return "(chat)"
}

func (c GraphChat) sender() string {
	return c.LastMessagePreview.From.User.DisplayName
}

func (c GraphChat) preview() string {
	body := c.LastMessagePreview.Body.Content
	if strings.EqualFold(c.LastMessagePreview.Body.ContentType, "html") {
		body = stripHTML(body)
	}
	body = strings.TrimSpace(body)
	const max = 80
	if len(body) > max {
		return body[:max] + "…"
	}
	return body
}

// stripHTML removes tags crudely — enough for a single-line preview.
func stripHTML(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '<':
			depth++
		case '>':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// fetchChats requests the user's chats with the latest-message preview and
// read viewpoint expanded so we can compute unread status client-side.
func fetchChats(ctx context.Context, token string) ([]GraphChat, error) {
	// Note: the space in $orderby must be percent-encoded for a valid request
	// target (slashes are kept as-is for the OData property path).
	orderby := strings.ReplaceAll("lastMessagePreview/createdDateTime desc", " ", "%20")
	u := graphBase + "/me/chats?$top=50&$expand=members,lastMessagePreview&$orderby=" + orderby
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graph chats request failed: %s", resp.Status)
	}

	var out struct {
		Value []GraphChat `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Value, nil
}
