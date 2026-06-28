package teams

import (
	"context"
	"strings"
	"time"
)

// readNotifs is the notification source, indirected so tests can inject one.
var readNotifs = readTeamsNotifications

// GetUnreadLocal reads recent Teams notifications from the operating system
// (macOS only) and presents them as "unread chats" — no API keys required. It
// groups by chat/sender, keeping the most recent message per conversation.
// nameFilter, if set, limits results to matching names (the favorites list).
// Workaround for unavailable azure details, does not work very stable!!!
func (s *Service) GetUnreadLocal(ctx context.Context, nameFilter []string) (Result, error) {
	notifs, err := readNotifs()
	if err != nil {
		return Result{}, err
	}

	var out Result
	seen := map[string]bool{}
	for _, n := range notifs { // newest first
		name := strings.TrimSpace(n.Title)
		if name == "" {
			name = "Teams"
		}
		if len(nameFilter) > 0 && !matchesAny(name, nameFilter) {
			continue
		}
		if seen[name] {
			continue // keep only the latest message per chat
		}
		seen[name] = true

		ts := ""
		if !n.Time.IsZero() {
			ts = n.Time.Format(time.RFC3339)
		}
		out.UnreadChats = append(out.UnreadChats, Chat{
			Name:      name,
			From:      strings.TrimSpace(n.Subtitle),
			Preview:   strings.TrimSpace(n.Body),
			Timestamp: ts,
		})
	}
	out.TotalUnread = len(out.UnreadChats)
	return out, nil
}
