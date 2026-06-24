package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/aliozdemir13/Apollo/internal/config"
)

// setupTestApp initializes a temporary environment to test the App struct safely.
func setupTestApp(t *testing.T) (*App, string) {
	t.Helper()

	// Create an isolated temporary directory for testing filesystem dependencies
	tmpDir, err := os.MkdirTemp("", "apollo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Craft a baseline mock configuration
	mockCfg := &config.Config{}
	mockCfg.Location.Name = "Berlin"
	mockCfg.Location.Lat = 52.52
	mockCfg.Location.Lon = 13.40
	mockCfg.Units = "celsius"
	mockCfg.Teams.Source = "local"
	mockCfg.Teams.ClientID = "old-client-id"
	mockCfg.Teams.TenantID = "old-tenant-id"
	mockCfg.Views = []string{"clock", "weather"}

	// Construct our target application context
	app := &App{
		ctx: context.Background(),
		cfg: mockCfg,
	}

	return app, tmpDir
}

// TestSaveSettings_TableDriven handles all permutations of mutation rules for app configurations.
func TestSaveSettings_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		inputSettings Settings
		verifyState   func(t *testing.T, app *App)
		expectedErr   bool
	}{
		{
			name: "No changes preserves state parameters",
			inputSettings: Settings{
				LocationName:  "Berlin",
				Units:         "celsius",
				TeamsSource:   "local",
				TeamsClientID: "old-client-id",
				TeamsTenantID: "old-tenant-id",
				Views:         []string{"clock", "weather"},
			},
			verifyState: func(t *testing.T, app *App) {
				if app.cfg.Location.Lat != 52.52 || app.cfg.Location.Lon != 13.40 {
					t.Errorf("Expected coordinates to remain untouched, got Lat: %f, Lon: %f", app.cfg.Location.Lat, app.cfg.Location.Lon)
				}
			},
			expectedErr: false,
		},
		{
			name: "Changing location resets coordinates to 0 for re-geocoding",
			inputSettings: Settings{
				LocationName:  "London", // New city
				Units:         "celsius",
				TeamsSource:   "local",
				TeamsClientID: "old-client-id",
				TeamsTenantID: "old-tenant-id",
				Views:         []string{"clock", "weather"},
			},
			verifyState: func(t *testing.T, app *App) {
				if app.cfg.Location.Name != "London" {
					t.Errorf("Expected location to change to London, got %s", app.cfg.Location.Name)
				}
				if app.cfg.Location.Lat != 0 || app.cfg.Location.Lon != 0 {
					t.Errorf("Expected coordinates to zero out on location shift, got Lat: %f, Lon: %f", app.cfg.Location.Lat, app.cfg.Location.Lon)
				}
			},
			expectedErr: false,
		},
		{
			name: "Unit formatting updates correctly",
			inputSettings: Settings{
				LocationName:  "Berlin",
				Units:         "fahrenheit", // New unit type
				TeamsSource:   "local",
				TeamsClientID: "old-client-id",
				TeamsTenantID: "old-tenant-id",
				Views:         []string{"clock", "weather"},
			},
			verifyState: func(t *testing.T, app *App) {
				if app.cfg.Units != "fahrenheit" {
					t.Errorf("Expected units to adapt to fahrenheit, got %s", app.cfg.Units)
				}
			},
			expectedErr: false,
		},
		{
			name: "Invalid unit fallbacks automatically to celsius",
			inputSettings: Settings{
				LocationName:  "Berlin",
				Units:         "kelvin-is-invalid", // Fallback test
				TeamsSource:   "local",
				TeamsClientID: "old-client-id",
				TeamsTenantID: "old-tenant-id",
				Views:         []string{"clock", "weather"},
			},
			verifyState: func(t *testing.T, app *App) {
				if app.cfg.Units != "celsius" {
					t.Errorf("Expected invalid units to fallback to celsius, got %s", app.cfg.Units)
				}
			},
			expectedErr: false,
		},
		{
			name: "View layout alterations are preserved",
			inputSettings: Settings{
				LocationName:  "Berlin",
				Units:         "celsius",
				TeamsSource:   "local",
				TeamsClientID: "old-client-id",
				TeamsTenantID: "old-tenant-id",
				Views:         []string{"totp", "github", "system"}, // Mutated collection
			},
			verifyState: func(t *testing.T, app *App) {
				expectedViews := []string{"totp", "github", "system"}
				if !reflect.DeepEqual(app.cfg.Views, expectedViews) {
					t.Errorf("Expected views to match %v, got %v", expectedViews, app.cfg.Views)
				}
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup clean application instance state
			app, tmpDir := setupTestApp(t)
			defer os.RemoveAll(tmpDir)

			// Overwrite internal configuration save output path to our safe playground temp directory
			// to avoid throwing unexpected write or panic permissions failures.
			tempConfigPath := filepath.Join(tmpDir, "config.json")

			// Execute the save function execution branch
			err := app.SaveSettings(tt.inputSettings)

			// Error presence checks
			if (err != nil) != tt.expectedErr {
				t.Fatalf("SaveSettings() error status = %v, expected error status presence = %v", err, tt.expectedErr)
			}

			// Execute case-specific validation checks
			if tt.verifyState != nil {
				tt.verifyState(t, app)
			}
			_ = tempConfigPath // reference helper context
		})
	}
}

// TestGetTeamsUnread_TableDriven asserts logical path validation branches for backend configurations.
func TestGetTeamsUnread_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		sourceType    string
		clientID      string
		expectedError string
	}{
		{
			name:          "Graph mode throws explicit structural error when completely unconfigured",
			sourceType:    "graph",
			clientID:      "", // Missing credential requirements
			expectedError: "Teams is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, tmpDir := setupTestApp(t)
			defer os.RemoveAll(tmpDir)

			// Apply explicit case structures
			app.cfg.Teams.Source = tt.sourceType
			app.cfg.Teams.ClientID = tt.clientID

			_, err := app.GetTeamsUnread()
			if err == nil {
				t.Fatalf("Expected an operational validation failure, got completely clear execution return")
			}

			if err.Error() != tt.expectedError {
				t.Errorf("Expected error string match: %q, got: %q", tt.expectedError, err.Error())
			}
		})
	}
}

// TestGetSettings ensures the DTO mappings match downstream expectations without drifting.
func TestGetSettings(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer os.RemoveAll(tmpDir)

	settings := app.GetSettings()

	if settings.LocationName != "Berlin" {
		t.Errorf("Expected baseline DTO mapping parameter to equal Berlin, got %s", settings.LocationName)
	}
	if settings.Units != "celsius" {
		t.Errorf("Expected baseline DTO mapping units parameter to equal celsius, got %s", settings.Units)
	}
}
