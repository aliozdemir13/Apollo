package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aliozdemir13/Apollo/internal/config"
	"github.com/aliozdemir13/Apollo/internal/sysstats"
	"github.com/aliozdemir13/Apollo/internal/totp"
	"github.com/aliozdemir13/Apollo/internal/weather"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// mfaUnlockWindow is how long the 2FA screen stays unlocked after a correct PIN
// before it auto-locks again.
const mfaUnlockWindow = 5 * time.Minute

// App is the single struct bound to the frontend. It owns the config and all
// service clients and exposes one method per data source.
type App struct {
	ctx     context.Context
	cfg     *config.Config
	weather *weather.Service
	sys     *sysstats.Service
	mfa     *totp.Service
}

// NewApp loads config and constructs the service clients.
func NewApp() *App {
	cfg, err := config.Load()
	if err != nil {
		// Fall back to in-memory defaults; the UI will surface the error later.
		fmt.Println("config load error:", err)
	}

	a := &App{
		cfg:     cfg,
		weather: weather.New(),
		sys:     sysstats.New(),
		mfa:     totp.New("Apollo"),
	}
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	clearNativeChrome() // strip the opaque/glass window backing on macOS
}

// domReady fires once the frontend has loaded; re-applies the transparent window
// backing in case the window wasn't ready at startup.
func (a *App) domReady(ctx context.Context) {
	clearNativeChrome()
}

// shutdown fires automatically when the user exits Apollo.
// This executes the kill switch we added to stop the background CPU sampler loop.
func (a *App) shutdown(ctx context.Context) {
	a.sys.Close() // this pushes the button that triggers s.stopChan!
}

// ---- Settings ---------------------------------------------------------------

// Settings is the flat DTO exchanged with the settings screen.
type Settings struct {
	LocationName string   `json:"locationName"`
	Units        string   `json:"units"`
	Views        []string `json:"views"`
	ConfigPath   string   `json:"configPath"`
}

// GetSettings returns the current configuration for the settings screen.
func (a *App) GetSettings() Settings {
	return Settings{
		LocationName: a.cfg.Location.Name,
		Units:        a.cfg.Units,
		Views:        a.cfg.Views,
		ConfigPath:   a.cfg.Path(),
	}
}

// SaveSettings applies and persists settings from the UI. Changing the location
// clears any cached coordinates so they are re-resolved on next fetch.
func (a *App) SaveSettings(s Settings) error {
	locationChanged := s.LocationName != a.cfg.Location.Name

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
	if len(s.Views) > 0 {
		a.cfg.Views = s.Views
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
			return weather.Data{}, err
		}
		lat, lon = geo.Latitude, geo.Longitude
		if name == "" {
			name = geo.Name
		}
		// Cache so we don't geocode every refresh.
		a.cfg.Location.Lat = lat
		a.cfg.Location.Lon = lon
		a.cfg.Location.Name = name
		_ = a.cfg.Save()
	}

	return a.weather.CurrentWeather(a.ctx, lat, lon, a.cfg.Units, name)
}

// ---- System stats -----------------------------------------------------------

// GetSystemStats returns a fresh CPU/RAM/battery snapshot.
func (a *App) GetSystemStats() (sysstats.Stats, error) {
	return a.sys.Get(a.ctx)
}

// GetTopProcesses returns the top CPU-consuming processes for the system view.
func (a *App) GetTopProcesses() ([]sysstats.Process, error) {
	return a.sys.TopProcesses(a.ctx)
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
	runtime.BrowserOpenURL(a.ctx, url)
}
