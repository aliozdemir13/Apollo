// Package config loads and persists user settings for the widget.
//
// Settings live in a single JSON file under the OS config dir:
//
//		macOS:  ~/Library/Application Support/Apollo/config.json
//		Linux:  ~/.config/Apollo/config.json
//	 	Windows: %APPDATA%\Apollo\config.json
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Location is the place weather is reported for. If Lat/Lon are both zero the
// backend will geocode Name (or fall back to IP-based detection).
type Location struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
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
	Units    string   `json:"units"` // "celsius" | "fahrenheit"
	MFA      MFA      `json:"mfa"`
	// Views is the ordered list of view ids the device cycles through.
	Views []string `json:"views"`
	// SeenViews records which canonical views the user has already been shown,
	// so newly released views surface once but stay disabled if later removed.
	SeenViews []string `json:"seenViews,omitempty"`

	mu   sync.Mutex `json:"-"`
	path string     `json:"-"`
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// AllViews is the canonical set of view ids understood by the frontend.
var AllViews = []string{"clock", "weather", "system", "totp"}

// Dir returns the directory the config file lives in, creating it if needed.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "Apollo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func defaults(path string) *Config {
	return &Config{
		Location:  Location{Name: ""},
		Units:     "celsius",
		Views:     append([]string(nil), AllViews...),
		SeenViews: append([]string(nil), AllViews...),
		path:      path,
	}
}

// Load reads the config file, returning sensible defaults if it does not exist.
func Load() (*Config, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "config.json")

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		c := defaults(path)
		_ = c.Save() // best-effort: write a starter file
		return c, nil
	}
	if err != nil {
		return nil, err
	}

	c := defaults(path)
	if err := json.Unmarshal(data, c); err != nil {
		return nil, err
	}
	c.path = path
	if len(c.Views) == 0 {
		c.Views = append([]string(nil), AllViews...)
	}
	if c.Units == "" {
		c.Units = "celsius"
	}

	// Surface views added in newer versions: any canonical view the user hasn't
	// seen yet is appended to their cycle once, then marked seen so removing it
	// later sticks.
	migrated := false
	for _, v := range AllViews {
		if !contains(c.SeenViews, v) {
			c.SeenViews = append(c.SeenViews, v)
			if !contains(c.Views, v) {
				c.Views = append(c.Views, v)
			}
			migrated = true
		}
	}
	if migrated {
		_ = c.Save()
	}
	return c, nil
}

// Save persists the config to disk atomically.
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}

// Path returns the absolute path of the config file (for display in the UI).
func (c *Config) Path() string { return c.path }
