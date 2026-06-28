// Package config loads and persists user settings for the widget.
//
// Settings live in a single JSON file under the OS config dir:
//
//	macOS:  ~/Library/Application Support/Apollo-Widget/config.json
//	Linux:  ~/.config/Apollo-Widget/config.json
//	Windows: %APPDATA%\Apollo-Widget\config.json
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// This variable is the "secret sauce".
// It defaults to the real OS call, but tests can swap it out.
var UserConfigDir = os.UserConfigDir

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// AllViews is the canonical set of view ids understood by the frontend.
var AllViews = []string{"clock", "weather", "system", "github", "teams", "totp"}

// Dir returns the directory the config file lives in, creating it if needed.
func Dir() (string, error) {
	// We call our variable here instead of os.UserConfigDir()
	base, err := UserConfigDir()

	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "Apollo-Widget")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func defaults(path string) *Config {
	// SeenViews is intentionally left nil: the migration in Load populates it.
	// Pre-filling it would make existing configs (which omit seenViews) appear
	// to have already seen every view, so newly added views would never surface.
	return &Config{
		Location: Location{Name: ""},
		Units:    "celsius",
		Theme:    "grey",
		Views:    append([]string(nil), AllViews...),
		path:     path,
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
	if c.Theme == "" {
		c.Theme = "grey"
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

	// 1. Ensure path is not empty (Prevents saving .tmp in the bin folder)
	if c.path == "" {
		dir, err := Dir()
		if err != nil {
			return err
		}
		c.path = filepath.Join(dir, "config.json")
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}

	// 2. Try to rename (Works perfectly on Mac/Linux)
	err = os.Rename(tmp, c.path)
	if err != nil {
		// 3. If Rename failed (Common on Windows), try deleting first
		// This block only really runs if the standard rename fails.
		_ = os.Remove(c.path)
		err = os.Rename(tmp, c.path)
	}

	return err
}

// Path returns the absolute path of the config file (for display in the UI).
func (c *Config) Path() string { return c.path }
