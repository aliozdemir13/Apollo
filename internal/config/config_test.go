package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Helper to force the config directory to a temp folder for the duration of the test
func setupTestConfig(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()

	// Save the original and restore after test
	old := UserConfigDir
	t.Cleanup(func() { UserConfigDir = old })

	// Override the global variable for this test
	UserConfigDir = func() (string, error) {
		return tmp, nil
	}
	return tmp
}

func TestContains(t *testing.T) {
	tests := []struct {
		list []string
		v    string
		want bool
	}{
		{[]string{"a", "b"}, "a", true},
		{[]string{"a", "b"}, "c", false},
		{nil, "a", false},
		{[]string{}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.v, func(t *testing.T) {
			if got := contains(tt.list, tt.v); got != tt.want {
				t.Errorf("contains(%v,%q)=%v want %v", tt.list, tt.v, got, tt.want)
			}
		})
	}
}

// writeConfig writes raw JSON to the config path under a redirected HOME.
func writeConfig(t *testing.T, json string) string {
	t.Helper()
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(json), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadFreshCreatesDefaults(t *testing.T) {
	setupTestConfig(t)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Views) != len(AllViews) {
		t.Errorf("views=%v want %v", c.Views, AllViews)
	}
	if c.Units != "celsius" {
		t.Errorf("units=%q", c.Units)
	}
	if c.Path() == "" {
		t.Error("path empty")
	}
	// The starter file should now exist on disk.
	if _, err := os.Stat(c.Path()); err != nil {
		t.Errorf("starter file not written: %v", err)
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	setupTestConfig(t)
	c, _ := Load()
	c.Location.Name = "Berlin"
	c.Units = "fahrenheit"
	c.GitHub.Token = "tok"
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	c2, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c2.Location.Name != "Berlin" || c2.Units != "fahrenheit" || c2.GitHub.Token != "tok" {
		t.Errorf("round-trip mismatch: %+v", c2)
	}
}

func TestLoadMigratesNewViews(t *testing.T) {
	setupTestConfig(t)
	// Old config predating the "totp" view, with no seenViews.
	writeConfig(t, `{"units":"celsius","views":["clock","weather"]}`)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !contains(c.Views, "totp") {
		t.Errorf("migration did not append totp: %v", c.Views)
	}
	for _, v := range AllViews {
		if !contains(c.SeenViews, v) {
			t.Errorf("seenViews missing %q: %v", v, c.SeenViews)
		}
	}
}

func TestLoadEmptyViewsAndUnitsDefaults(t *testing.T) {
	setupTestConfig(t)
	writeConfig(t, `{"views":[],"units":"","seenViews":["clock","weather","system","github","teams","totp"]}`)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Views) != len(AllViews) {
		t.Errorf("empty views not defaulted: %v", c.Views)
	}
	if c.Units != "celsius" {
		t.Errorf("units=%q want celsius", c.Units)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	setupTestConfig(t)
	writeConfig(t, `{not valid json`)
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestDir(t *testing.T) {
	setupTestConfig(t)
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("dir not created: %v", err)
	}
}

func TestDirAndLoadErrorWhenNoHome(t *testing.T) {
	// Save and restore
	old := UserConfigDir
	t.Cleanup(func() { UserConfigDir = old })

	// Explicitly force an error from the config dir lookup
	UserConfigDir = func() (string, error) {
		return "", os.ErrNotExist
	}

	if _, err := Dir(); err == nil {
		t.Error("Dir should error when UserConfigDir returns an error")
	}
	if _, err := Load(); err == nil {
		t.Error("Load should error when UserConfigDir returns an error")
	}
}

func TestDirMkdirError(t *testing.T) {
	old := UserConfigDir
	t.Cleanup(func() { UserConfigDir = old })

	// Create a file where a directory should be to force a MkdirAll failure
	tmp := t.TempDir()
	blockedPath := filepath.Join(tmp, "blocked")
	os.WriteFile(blockedPath, []byte("i am a file"), 0644)

	// Mock UserConfigDir to return a path that cannot have subdirectories created
	UserConfigDir = func() (string, error) {
		return blockedPath, nil
	}

	if _, err := Dir(); err == nil {
		t.Error("Dir should error when it cannot create the Apollo-Widget subdirectory")
	}
}

func TestSaveWriteError(t *testing.T) {
	// Path under a non-existent directory → WriteFile fails.
	c := defaults("/no/such/dir/Apollo-Widget/config.json")
	if err := c.Save(); err == nil {
		t.Error("Save should error writing to a missing directory")
	}
}

func TestSaveRenameError(t *testing.T) {
	// path is an existing directory → the temp file writes, but Rename onto a
	// directory fails.
	target := filepath.Join(t.TempDir(), "asdir")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	c := defaults(target)
	if err := c.Save(); err == nil {
		t.Error("Save should error renaming onto a directory")
	}
}

func TestLoadReadError(t *testing.T) {
	setupTestConfig(t)
	dir, _ := Dir()
	// Make config.json a directory so ReadFile fails with a non-NotExist error.
	if err := os.Mkdir(filepath.Join(dir, "config.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Error("Load should error when config path is unreadable")
	}
}
