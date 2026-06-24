package config

import (
	"os"
	"path/filepath"
	"testing"
)

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
	t.Setenv("HOME", t.TempDir())
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
	t.Setenv("HOME", t.TempDir())
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
	t.Setenv("HOME", t.TempDir())
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
	t.Setenv("HOME", t.TempDir())
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
	t.Setenv("HOME", t.TempDir())
	writeConfig(t, `{not valid json`)
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("dir not created: %v", err)
	}
}

func TestDirAndLoadErrorWhenNoHome(t *testing.T) {
	t.Setenv("HOME", "") // os.UserConfigDir errors without $HOME on unix
	if _, err := Dir(); err == nil {
		t.Error("Dir should error without HOME")
	}
	if _, err := Load(); err == nil {
		t.Error("Load should error without HOME")
	}
}

func TestSaveWriteError(t *testing.T) {
	// Path under a non-existent directory → WriteFile fails.
	c := defaults("/no/such/dir/Apollo-Widget/config.json")
	if err := c.Save(); err == nil {
		t.Error("Save should error writing to a missing directory")
	}
}

func TestDirMkdirError(t *testing.T) {
	// HOME points at a regular file, so MkdirAll under it fails (not a directory).
	f := filepath.Join(t.TempDir(), "homefile")
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", f)
	if _, err := Dir(); err == nil {
		t.Error("Dir should error when HOME is a file")
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
	t.Setenv("HOME", t.TempDir())
	dir, _ := Dir()
	// Make config.json a directory so ReadFile fails with a non-NotExist error.
	if err := os.Mkdir(filepath.Join(dir, "config.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Error("Load should error when config path is unreadable")
	}
}
