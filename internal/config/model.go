package config

import "sync"

// Location is the place weather is reported for. If Lat/Lon are both zero the
// backend will geocode Name (or fall back to IP-based detection).
type Location struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

// GitHub holds the credentials and repo selection for the PR view.
type GitHub struct {
	Token string   `json:"token"`           // personal access token (repo scope)
	Repos []string `json:"repos"`           // "owner/name" entries
	Login string   `json:"login,omitempty"` // optional: filter to PRs authored by / assigned to
}

// Teams holds the Azure AD app registration used for the Microsoft Graph
// device-code flow. No client secret is needed for a public client.
type Teams struct {
	// Source selects where unread data comes from: "graph" (Microsoft Graph,
	// needs an Azure app) or "local" (macOS Notification Center, needs Full Disk
	// Access, no keys). Empty defaults to "graph".
	Source   string `json:"source"`
	ClientID string `json:"clientId"`
	TenantID string `json:"tenantId"` // "common", "organizations", or a tenant GUID
	// Favorites is an optional allow-list of chat-name substrings. When set,
	// only matching chats are shown (emulating the Teams "favorites" section).
	Favorites []string `json:"favorites"`
}

// TotpAccount is non-secret metadata for one TOTP entry (e.g. a Salesforce
// org). The secret itself lives in the OS keychain keyed by ID, never here.
type TotpAccount struct {
	ID     string `json:"id"`
	Label  string `json:"label"`  // user-facing name, e.g. "ACME Prod"
	Issuer string `json:"issuer"` // e.g. "Salesforce"
}

// MFA holds the list of TOTP accounts. The PIN hash is stored in the keychain.
type MFA struct {
	Accounts []TotpAccount `json:"accounts"`
}

// Config is the full persisted settings document.
type Config struct {
	Location Location `json:"location"`
	Units    string   `json:"units"`           // "celsius" | "fahrenheit"
	Theme    string   `json:"theme,omitempty"` // widget colour theme id; empty = "grey"
	GitHub   GitHub   `json:"github"`
	Teams    Teams    `json:"teams"`
	MFA      MFA      `json:"mfa"`
	// Views is the ordered list of view ids the device cycles through.
	Views []string `json:"views"`
	// SeenViews records which canonical views the user has already been shown,
	// so newly released views surface once but stay disabled if later removed.
	SeenViews []string `json:"seenViews,omitempty"`

	mu   sync.Mutex `json:"-"`
	path string     `json:"-"`
}
