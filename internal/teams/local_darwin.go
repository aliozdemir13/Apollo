//go:build darwin

package teams

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"howett.net/plist"
	_ "modernc.org/sqlite"
)

const teamsBundleID = "com.microsoft.teams2"

// readTeamsNotifications reads delivered Teams notifications from the macOS
// Notification Center database. The file is TCC-protected, so the app must be
// granted Full Disk Access. Opened read-only + immutable to avoid lock issues
// while the notification daemon is writing.
func readTeamsNotifications() ([]Notif, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(home, "Library/Group Containers/group.com.apple.usernoted/db2/db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil, friendly(err)
	}

	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro&immutable=1")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT r.data
		FROM record r
		JOIN app a ON a.app_id = r.app_id
		WHERE a.identifier = ?
		ORDER BY r.rec_id DESC
		LIMIT 60`, teamsBundleID)
	if err != nil {
		return nil, friendly(err)
	}
	defer rows.Close()

	var out []Notif
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			continue
		}
		if n, ok := parseNotif(data); ok {
			out = append(out, n)
		}
	}
	return out, rows.Err()
}

// friendly converts the opaque TCC errors into actionable guidance.
func friendly(err error) error {
	m := strings.ToLower(err.Error())
	if strings.Contains(m, "not permitted") ||
		strings.Contains(m, "authorization denied") ||
		strings.Contains(m, "unable to open") {
		return fmt.Errorf("grant Full Disk Access (System Settings > Privacy)")
	}
	return err
}

// parseNotif decodes a notification's binary-plist blob. The schema varies by
// macOS version, so we walk the whole structure and pick up the title/subtitle/
// body/date fields wherever they appear.
func parseNotif(data []byte) (Notif, bool) {
	var top interface{}
	if _, err := plist.Unmarshal(data, &top); err != nil {
		return Notif{}, false
	}

	var n Notif
	walk(top, func(m map[string]interface{}) {
		if n.Title == "" {
			if s, ok := m["titl"].(string); ok {
				n.Title = s
			}
		}
		if n.Subtitle == "" {
			if s, ok := m["subt"].(string); ok {
				n.Subtitle = s
			}
		}
		if n.Body == "" {
			if s, ok := m["body"].(string); ok {
				n.Body = s
			}
		}
		if n.Time.IsZero() {
			if t, ok := m["date"].(time.Time); ok {
				n.Time = t
			}
		}
	})

	if n.Title == "" && n.Body == "" {
		return Notif{}, false
	}
	return n, true
}

// walk visits every map within a decoded plist structure.
func walk(v interface{}, fn func(map[string]interface{})) {
	switch t := v.(type) {
	case map[string]interface{}:
		fn(t)
		for _, val := range t {
			walk(val, fn)
		}
	case []interface{}:
		for _, val := range t {
			walk(val, fn)
		}
	}
}
