package teams

import "time"

// DeviceCode carries the instructions the user must follow to sign in.
type DeviceCode struct {
	UserCode        string `json:"userCode"`
	VerificationURL string `json:"verificationUrl"`
	Message         string `json:"message"`
}

// Chat is one chat with an unread message.
type Chat struct {
	ID        string `json:"id"`
	Name      string `json:"name"`      // topic, or the other person's name
	Preview   string `json:"preview"`   // snippet of the latest message
	From      string `json:"from"`      // sender display name
	Timestamp string `json:"timestamp"` // RFC3339 of latest message
}

// Result is the unread summary returned to the frontend.
type Result struct {
	UnreadChats []Chat `json:"unreadChats"`
	TotalUnread int    `json:"totalUnread"`
	NeedsLogin  bool   `json:"needsLogin"`
}

// Notif is one OS notification surfaced by the platform reader.
type Notif struct {
	Title    string
	Subtitle string
	Body     string
	Time     time.Time
}

// GraphChat mirrors the subset of the Graph chat resource we use.
type GraphChat struct {
	ID        string `json:"id"`
	Topic     string `json:"topic"`
	ChatType  string `json:"chatType"`
	Viewpoint struct {
		LastMessageReadDateTime string `json:"lastMessageReadDateTime"`
	} `json:"viewpoint"`
	LastMessagePreview struct {
		CreatedDateTime string `json:"createdDateTime"`
		Body            struct {
			Content     string `json:"content"`
			ContentType string `json:"contentType"`
		} `json:"body"`
		From struct {
			User struct {
				DisplayName string `json:"displayName"`
			} `json:"user"`
		} `json:"from"`
	} `json:"lastMessagePreview"`
	Members []struct {
		DisplayName string `json:"displayName"`
	} `json:"members"`
}
