package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aliozdemir13/Apollo/internal/config"
	"github.com/aliozdemir13/Apollo/internal/github"
	"github.com/aliozdemir13/Apollo/internal/sysstats"
	"github.com/aliozdemir13/Apollo/internal/teams"
	"github.com/aliozdemir13/Apollo/internal/totp"
	"github.com/aliozdemir13/Apollo/internal/weather"
)

// setupMockServer creates a local web server that mimics GitHub and Open-Meteo APIs.
func setupMockServer(t *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Routing logic to return different JSON based on the URL
		switch {
		case strings.Contains(r.URL.Path, "/v1/search"): // Weather Geocode
			w.Write([]byte(`{"results": [{"name": "Berlin", "latitude": 52.52, "longitude": 13.40}]}`))
		case strings.Contains(r.URL.Path, "/v1/forecast"): // Weather Forecast
			w.Write([]byte(`{"current": {"temperature_2m": 22.5, "weather_code": 0}}`))
		case strings.Contains(r.URL.Path, "/repos/") && strings.Contains(r.URL.Path, "/pulls"): // GitHub PRs
			w.Write([]byte(`[{"number": 1, "title": "Test PR", "user": {"login": "ali"}, "html_url": "http://gh.com/1"}]`))
		case strings.Contains(r.URL.Path, "/search/issues"): // GitHub Reviews
			w.Write([]byte(`{"items": [{"number": 2, "title": "Review Me", "repository_url": "http://api.gh.com/repos/owner/repo"}]}`))
		case strings.Contains(r.URL.Path, "/me/chats"): // Teams Graph API
			w.Write([]byte(`{"value": []}`))
		default:
			w.Write([]byte(`{}`))
		}
	}))

	// Redirect internal package URLs to the mock server
	oldWeather := weather.ForecastBase
	oldGeo := weather.GeocodeBase
	oldGH := github.ApiBase // Uncomment if you exported this

	weather.ForecastBase = server.URL + "/v1/forecast"
	weather.GeocodeBase = server.URL + "/v1/search"
	github.ApiBase = server.URL

	t.Cleanup(func() {
		server.Close()
		weather.ForecastBase = oldWeather
		weather.GeocodeBase = oldGeo
		github.ApiBase = oldGH
	})

	return server

}

// Helper to force the config directory to a temp folder for the duration of the test
func setupTestConfig(t *testing.T) string {
	t.Helper()

	// Create a unique temp directory for this specific test run
	// This prevents parallel tests from clobbering each other
	tmpDir := t.TempDir()

	// Save original and restore after test
	old := config.UserConfigDir
	t.Cleanup(func() { config.UserConfigDir = old })

	// Override the global variable
	config.UserConfigDir = func() (string, error) {
		return tmpDir, nil
	}

	// The config package appends "Apollo-Widget", so we return the full path
	// where config.json will actually live.
	return filepath.Join(tmpDir, "Apollo-Widget")
}

// setupTestApp initializes a temporary environment to test the App struct safely.
// setupTestApp initializes a temporary environment to test the App struct safely.
func setupTestApp(t *testing.T) (*App, string) {
	t.Helper()

	// Redirect config to a temp folder
	configPathDir := setupTestConfig(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	// Override defaults with stable test values
	cfg.Location.Name = "Berlin"
	cfg.Location.Lat = 52.52
	cfg.Location.Lon = 13.40
	cfg.Units = "celsius"
	cfg.Teams.Source = "local"
	cfg.Teams.ClientID = "old-client-id"
	cfg.Teams.TenantID = "old-tenant-id"
	cfg.Views = []string{"clock", "weather"}

	// FIX: Initialize ALL services so app.go methods don't hit nil pointers
	app := &App{
		ctx:     context.Background(),
		cfg:     cfg,
		weather: weather.New(),           // <--- Added
		sys:     sysstats.New(),          // <--- Added
		gh:      github.New(),            // <--- Added
		mfa:     totp.New("Apollo-Test"), // <--- Added
		teams:   teams.New(cfg.Teams.ClientID, cfg.Teams.TenantID, filepath.Join(configPathDir, "teams_token.json")),
	}

	// Add a cleanup to stop the sysstats background loop
	t.Cleanup(func() {
		app.sys.Close()
	})

	return app, configPathDir
}

func TestAppLifecycle(t *testing.T) {
	setupTestConfig(t) // Isolate config
	app := NewApp()

	// Test startup/domReady (these call clearNativeChrome which is a no-op on non-mac)
	app.startup(context.Background())
	app.domReady(context.Background())

	// Test views
	views := app.Views()
	if len(views) == 0 {
		t.Error("Expected default views to be populated")
	}

	// Test shutdown (closes the sysstats collector)
	app.shutdown(context.Background())
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
			name: "Valid theme is persisted",
			inputSettings: Settings{
				LocationName:  "Berlin",
				Units:         "celsius",
				Theme:         "purple",
				TeamsSource:   "local",
				TeamsClientID: "old-client-id",
				TeamsTenantID: "old-tenant-id",
				Views:         []string{"clock", "weather"},
			},
			verifyState: func(t *testing.T, app *App) {
				if app.cfg.Theme != "purple" {
					t.Errorf("Expected theme purple, got %s", app.cfg.Theme)
				}
			},
			expectedErr: false,
		},
		{
			name: "Invalid theme falls back to grey",
			inputSettings: Settings{
				LocationName:  "Berlin",
				Units:         "celsius",
				Theme:         "chartreuse-is-invalid",
				TeamsSource:   "local",
				TeamsClientID: "old-client-id",
				TeamsTenantID: "old-tenant-id",
				Views:         []string{"clock", "weather"},
			},
			verifyState: func(t *testing.T, app *App) {
				if app.cfg.Theme != "grey" {
					t.Errorf("Expected invalid theme to fall back to grey, got %s", app.cfg.Theme)
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

			// Apply explicit case structures and sync the teams service
			app.cfg.Teams.Source = tt.sourceType
			app.cfg.Teams.ClientID = tt.clientID
			app.teams = teams.New(tt.clientID, "", tmpDir+"/cache")

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

func TestAppFullCoverage(t *testing.T) {
	app, _ := setupTestApp(t)
	setupMockServer(t)

	// Inject the mock URL into services that allow it (assuming hooks exist)
	// If your internal packages don't export these, see "Pro-Tip" below.

	t.Run("Lifecycle", func(t *testing.T) {
		app.startup(context.Background())
		app.domReady(context.Background())
		app.shutdown(context.Background())
	})

	t.Run("Weather Pass-through", func(t *testing.T) {
		app.cfg.Location.Lat, app.cfg.Location.Lon = 0, 0 // Force geocode path
		_, _ = app.GetWeather()
	})

	t.Run("GitHub Pass-through", func(t *testing.T) {
		app.cfg.GitHub.Token = "fake-token"
		app.cfg.GitHub.Repos = []string{"owner/repo"}
		_, _ = app.GetGitHubPRs()
		_, _ = app.GetGitHubReviews()
		_, _ = app.GetGitHubWorkflows()
	})

	t.Run("System Stats Pass-through", func(t *testing.T) {
		_, _ = app.GetSystemStats()
		_, _ = app.GetTopProcesses()
	})

	t.Run("MFA Logic", func(t *testing.T) {
		_ = app.MFASetPin("1234")
		app.MFAUnlock("1234")
		app.MFAGetStatus()
		app.MFAGetCodes()
		app.MFALock()
	})

	t.Run("Settings Management", func(t *testing.T) {
		s := app.GetSettings()
		s.Theme = "purple"
		_ = app.SaveSettings(s)

		if app.cfg.Theme != "purple" {
			t.Errorf("Theme not saved, got %s", app.cfg.Theme)
		}
	})
}

func TestTeamsSourceSwitching(t *testing.T) {
	app, _ := setupTestApp(t)

	// Test Local Source
	app.cfg.Teams.Source = "local"
	_, _ = app.GetTeamsUnread()

	// Test Graph Source (Unconfigured)
	app.cfg.Teams.Source = "graph"
	app.cfg.Teams.ClientID = ""

	// CRITICAL: Rebuild the service to recognize the empty ClientID
	app.rebuildTeams()

	_, err := app.GetTeamsUnread()
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Errorf("Expected configuration error for Graph source, got: %v", err)
	}
}

func TestOpenURL(t *testing.T) {
	app, _ := setupTestApp(t)
	// This calls wails runtime.BrowserOpenURL, which is safe in tests (it does nothing)
	app.OpenURL("http://google.com")
}
