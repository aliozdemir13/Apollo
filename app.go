package main

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/aliozdemir13/Apollo/internal/config"
	"github.com/aliozdemir13/Apollo/internal/github"
	"github.com/aliozdemir13/Apollo/internal/sysstats"
	"github.com/aliozdemir13/Apollo/internal/teams"
	"github.com/aliozdemir13/Apollo/internal/totp"
	"github.com/aliozdemir13/Apollo/internal/weather"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// mfaUnlockWindow is how long the 2FA screen stays unlocked after a correct PIN
// before it auto-locks again.
const mfaUnlockWindow = 2 * time.Minute

// App is the single struct bound to the frontend. It owns the config and all
// service clients and exposes one method per data source.
type App struct {
	ctx     context.Context
	cfg     *config.Config
	weather *weather.Service
	sys     *sysstats.Service
	gh      *github.Service
	teams   *teams.Service
	mfa     *totp.Service
}

// NewApp loads config and constructs the service clients.
func NewApp() *App {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load error", "err", err)
		// so a.cfg is not nil, preventing the crash.
		// issue happend on the windows build
		cfg = &config.Config{}
	}

	a := &App{
		cfg:     cfg,
		weather: weather.New(),
		sys:     sysstats.New(),
		gh:      github.New(),
		mfa:     totp.New("Apollo-Widget"),
	}
	a.rebuildTeams()
	return a
}

func (a *App) rebuildTeams() {
	dir, _ := config.Dir()
	cachePath := filepath.Join(dir, "teams_token.json")
	a.teams = teams.New(a.cfg.Teams.ClientID, a.cfg.Teams.TenantID, cachePath)
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	clearNativeChrome() // strip the opaque/glass window backing on macOS
}

// domReady fires once the frontend has loaded; re-applies the transparent window
// backing in case the window wasn't ready at startup.
// added due to a bug causing the window still being visible on macOS after the app is started,
// but before the frontend is ready. This caused the window to be visible with a gray background and a shadow,
// which is not desired.
func (a *App) domReady(ctx context.Context) {
	clearNativeChrome()
}

// shutdown fires automatically when the user exits Apollo.
// This executes the kill switch we added to stop the background CPU sampler loop.
func (a *App) shutdown(ctx context.Context) {
	a.sys.Close() // this pushes the button that triggers s.stopChan, to avoid data leaks and goroutine leaks.
}

// ---- Settings ---------------------------------------------------------------

// validThemes is the set of widget colour theme ids the frontend can render.
// Keep in sync with the [data-theme="…"] blocks in frontend/src/style.css.
var validThemes = map[string]bool{
	"grey":    true,
	"purple":  true, // Purple — purple body, gold button
	"indigo":  true, // Indigo — slate body, indigo button
	"black":   true, // Black — black body, default buttons
	"blue":    true, // Blue — blue body, orange button
	"emerald": true, // Emerald — emerald-green body, gold button
	"red":     true, // Red — deep-red body, silver button
}

// Settings is the flat DTO exchanged with the settings screen.
type Settings struct {
	LocationName   string   `json:"locationName"`
	Units          string   `json:"units"`
	Theme          string   `json:"theme"`
	GithubToken    string   `json:"githubToken"`
	GithubRepos    []string `json:"githubRepos"`
	GithubLogin    string   `json:"githubLogin"`
	TeamsSource    string   `json:"teamsSource"`
	TeamsClientID  string   `json:"teamsClientId"`
	TeamsTenantID  string   `json:"teamsTenantId"`
	TeamsFavorites []string `json:"teamsFavorites"`
	Views          []string `json:"views"`
	ConfigPath     string   `json:"configPath"`
}

// GetSettings returns the current configuration for the settings screen.
func (a *App) GetSettings() Settings {
	return Settings{
		LocationName:   a.cfg.Location.Name,
		Units:          a.cfg.Units,
		Theme:          a.cfg.Theme,
		GithubToken:    a.cfg.GitHub.Token,
		GithubRepos:    a.cfg.GitHub.Repos,
		GithubLogin:    a.cfg.GitHub.Login,
		TeamsSource:    a.cfg.Teams.Source,
		TeamsClientID:  a.cfg.Teams.ClientID,
		TeamsTenantID:  a.cfg.Teams.TenantID,
		TeamsFavorites: a.cfg.Teams.Favorites,
		Views:          a.cfg.Views,
		ConfigPath:     a.cfg.Path(),
	}
}

// SaveSettings applies and persists settings from the UI. Changing the location
// clears any cached coordinates so they are re-resolved on next fetch.
func (a *App) SaveSettings(s Settings) error {
	locationChanged := s.LocationName != a.cfg.Location.Name
	teamsChanged := s.TeamsClientID != a.cfg.Teams.ClientID || s.TeamsTenantID != a.cfg.Teams.TenantID

	a.cfg.Location.Name = s.LocationName
	if locationChanged {
		a.cfg.Location.Lat = 0
		a.cfg.Location.Lon = 0
	}
	if s.Units == "fahrenheit" {
		a.cfg.Units = "fahrenheit"
	} else {
		a.cfg.Units = "celsius"
	}
	if validThemes[s.Theme] {
		a.cfg.Theme = s.Theme
	} else {
		a.cfg.Theme = "grey"
	}
	a.cfg.GitHub.Token = s.GithubToken
	a.cfg.GitHub.Repos = s.GithubRepos
	a.cfg.GitHub.Login = s.GithubLogin
	a.cfg.Teams.Source = s.TeamsSource
	a.cfg.Teams.ClientID = s.TeamsClientID
	a.cfg.Teams.TenantID = s.TeamsTenantID
	a.cfg.Teams.Favorites = s.TeamsFavorites
	if len(s.Views) > 0 {
		a.cfg.Views = s.Views
	}

	if teamsChanged {
		a.rebuildTeams()
	}
	return a.cfg.Save()
}

// Views returns the ordered list of enabled view ids.
func (a *App) Views() []string {
	return a.cfg.Views
}

// ---- Weather ----------------------------------------------------------------

// GetWeather resolves the configured location (geocoding or IP detection on
// first use, then caching the coordinates) and returns current conditions.
func (a *App) GetWeather() (weather.Data, error) {
	lat, lon := a.cfg.Location.Lat, a.cfg.Location.Lon
	name := a.cfg.Location.Name

	if lat == 0 && lon == 0 {
		var geo weather.GeoResult
		var err error
		if name != "" {
			geo, err = a.weather.Geocode(a.ctx, name)
		} else {
			geo, err = a.weather.DetectLocation(a.ctx)
		}
		if err != nil {
			// error logging is done in the weather package level
			return weather.Data{}, err
		}
		lat, lon = geo.Latitude, geo.Longitude
		if name == "" {
			name = geo.Name
		}
		// Cache so we don't geocode every refresh.
		// this is mainly for avoiding open source projects used in the weather package
		// and to not harrass the systems too frequenctly
		a.cfg.Location.Lat = lat
		a.cfg.Location.Lon = lon
		a.cfg.Location.Name = name
		_ = a.cfg.Save()
	}

	return a.weather.CurrentWeather(a.ctx, lat, lon, a.cfg.Units, name)
}

// ---- System stats -----------------------------------------------------------

// GetSystemStats returns a fresh CPU/RAM/battery snapshot.
// Each platform has its own implementation of sysstats.Service
func (a *App) GetSystemStats() (sysstats.Stats, error) {
	return a.sys.Get(a.ctx)
}

// GetTopProcesses returns the top CPU-consuming processes for the system view.
// Each platform has its own implementation of sysstats.Service
func (a *App) GetTopProcesses() ([]sysstats.Process, error) {
	return a.sys.TopProcesses(a.ctx)
}

// ---- GitHub -----------------------------------------------------------------

// GetGitHubPRs returns open PRs across the configured repos.
func (a *App) GetGitHubPRs() (github.Result, error) {
	return a.gh.GetPRs(a.ctx, a.cfg.GitHub.Token, a.cfg.GitHub.Repos, a.cfg.GitHub.Login)
}

// GetGitHubReviews returns open PRs awaiting my review across the configured repos.
func (a *App) GetGitHubReviews() (github.Result, error) {
	return a.gh.GetReviewRequests(a.ctx, a.cfg.GitHub.Token, a.cfg.GitHub.Repos)
}

// GetGitHubWorkflows returns the latest Actions run per configured repo.
func (a *App) GetGitHubWorkflows() (github.WorkflowResult, error) {
	return a.gh.GetWorkflowRuns(a.ctx, a.cfg.GitHub.Token, a.cfg.GitHub.Repos)
}

// ---- Teams ------------------------------------------------------------------

// GetTeamsUnread returns unread chats from the configured source.
func (a *App) GetTeamsUnread() (teams.Result, error) {
	if a.cfg.Teams.Source == "local" { // only functional in macos
		return a.teams.GetUnreadLocal(a.ctx, a.cfg.Teams.Favorites)
	}
	if !a.teams.Configured() {
		return teams.Result{}, fmt.Errorf("Teams is not configured")
	}
	return a.teams.GetUnread(a.ctx, a.cfg.Teams.Favorites)
}

// TeamsConfigured reports whether an Azure client ID has been set.
func (a *App) TeamsConfigured() bool { return a.teams.Configured() }

// TeamsLoggedIn reports whether a cached token is available.
func (a *App) TeamsLoggedIn() bool { return a.teams.LoggedIn(a.ctx) }

// TeamsLogin starts the device-code flow. The device code is emitted to the
// frontend via the "teams:devicecode" event; this call resolves once sign-in
// completes (or errors).
// Inside app.go -> TeamsLogin()
func (a *App) TeamsLogin() {
	go func() {
		slog.Info("teams login flow started")
		err := a.teams.Login(a.ctx, func(dc teams.DeviceCode) {
			// Force lowerCamelCase keys to match your TypeScript/Svelte declarations perfectly
			payload := map[string]string{
				"userCode":        dc.UserCode,
				"verificationUrl": dc.VerificationURL,
				"message":         dc.Message,
			}
			runtime.EventsEmit(a.ctx, "teams:device_code", payload)

			// Open the browser
			// TODO: currently there is an issue on the callback to finalize authentication
			runtime.BrowserOpenURL(a.ctx, dc.VerificationURL)
		})

		if err != nil {
			slog.Error("teams login error", "err", err)
			runtime.EventsEmit(a.ctx, "teams:login_error", err.Error())
			return
		}

		slog.Info("teams login complete")
		runtime.EventsEmit(a.ctx, "teams:auth_complete", true)
	}()
}

// ---- MFA / TOTP -------------------------------------------------------------

// MFAStatus summarises the 2FA screen state for the frontend.
type MFAStatus struct {
	HasPin           bool `json:"hasPin"`
	Unlocked         bool `json:"unlocked"`
	AccountCount     int  `json:"accountCount"`
	SecondsUntilLock int  `json:"secondsUntilLock"`
}

// MFACodeEntry is a single account's live code.
type MFACodeEntry struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Issuer  string `json:"issuer"`
	Code    string `json:"code"`
	Seconds int    `json:"seconds"` // seconds remaining in the 30s window
	Error   string `json:"error,omitempty"`
}

// MFACodes is the payload for the unlocked 2FA screen.
type MFACodes struct {
	Locked           bool           `json:"locked"`
	Entries          []MFACodeEntry `json:"entries"`
	SecondsUntilLock int            `json:"secondsUntilLock"`
}

// MFAGetStatus reports PIN/lock/account state (safe to call while locked).
func (a *App) MFAGetStatus() MFAStatus {
	return MFAStatus{
		HasPin:           a.mfa.HasPin(),
		Unlocked:         a.mfa.Unlocked(),
		AccountCount:     len(a.cfg.MFA.Accounts),
		SecondsUntilLock: a.mfa.SecondsUntilLock(),
	}
}

// MFAListAccounts returns account metadata (no secrets).
func (a *App) MFAListAccounts() []config.TotpAccount {
	if a.cfg.MFA.Accounts == nil {
		return []config.TotpAccount{}
	}
	return a.cfg.MFA.Accounts
}

// MFAAddAccount stores a new TOTP account's secret in the keychain and records
// its metadata. issuer defaults to "Salesforce".
// IT IS IMPORTANT TO NOTE THAT THIS FUNCTION AIMS CONVENIENCE OVER SECURITY, TREAT ACCORDINGLY.
// The secret is stored in the OS keychain, which is not encrypted by a user password.
// If you need to store secrets securely, consider using a dedicated password manager or hardware token.
func (a *App) MFAAddAccount(label, issuer, secret string) (config.TotpAccount, error) {
	if issuer == "" {
		issuer = "Salesforce"
	}
	acct := config.TotpAccount{ID: uuid.NewString(), Label: label, Issuer: issuer}
	if err := a.mfa.SetSecret(acct.ID, secret); err != nil {
		return config.TotpAccount{}, err
	}
	a.cfg.MFA.Accounts = append(a.cfg.MFA.Accounts, acct)
	if err := a.cfg.Save(); err != nil {
		_ = a.mfa.DeleteSecret(acct.ID) // roll back the secret on save failure
		return config.TotpAccount{}, err
	}
	return acct, nil
}

// MFARemoveAccount deletes an account's secret and metadata.
func (a *App) MFARemoveAccount(id string) error {
	_ = a.mfa.DeleteSecret(id)
	kept := a.cfg.MFA.Accounts[:0]
	for _, acct := range a.cfg.MFA.Accounts {
		if acct.ID != id {
			kept = append(kept, acct)
		}
	}
	a.cfg.MFA.Accounts = kept
	return a.cfg.Save()
}

// MFASetPin sets or replaces the unlock PIN.
func (a *App) MFASetPin(pin string) error { return a.mfa.SetPin(pin) }

// MFAClearPin removes the PIN (and locks the screen).
func (a *App) MFAClearPin() error { return a.mfa.ClearPin() }

// MFAUnlock opens the access window if the PIN is correct.
func (a *App) MFAUnlock(pin string) bool { return a.mfa.Unlock(pin, mfaUnlockWindow) }

// MFALock closes the access window immediately.
func (a *App) MFALock() { a.mfa.Lock() }

// MFAGetCodes returns live codes when unlocked, otherwise a locked marker.
func (a *App) MFAGetCodes() MFACodes {
	if !a.mfa.Unlocked() {
		return MFACodes{Locked: true}
	}
	out := MFACodes{SecondsUntilLock: a.mfa.SecondsUntilLock()}
	for _, acct := range a.cfg.MFA.Accounts {
		e := MFACodeEntry{ID: acct.ID, Label: acct.Label, Issuer: acct.Issuer}
		code, secs, err := a.mfa.Code(acct.ID)
		if err != nil {
			e.Error = err.Error()
		} else {
			e.Code = code
			e.Seconds = secs
		}
		out.Entries = append(out.Entries, e)
	}
	return out
}

// ---- Misc -------------------------------------------------------------------

// OpenURL opens a link in the user's default browser (used for PR links).
func (a *App) OpenURL(url string) {
	// If ctx is nil or Background (common in tests), don't call Wails runtime
	slog.Info("context type: %T", "ctx", fmt.Sprintf("%T", a.ctx))
	if a.ctx == nil || fmt.Sprintf("%T", a.ctx) == "context.backgroundCtx" {
		slog.Info("Skipping BrowserOpenURL in test environment", "url", url)
		return
	}
	runtime.BrowserOpenURL(a.ctx, url)
}
