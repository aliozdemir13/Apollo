//go:build !darwin

package teams

import "fmt"

// readTeamsNotifications is macOS-only; other platforms have no equivalent
// persisted notification store to read.
func readTeamsNotifications() ([]Notif, error) {
	return nil, fmt.Errorf("local Teams source is only available on macOS")
}
