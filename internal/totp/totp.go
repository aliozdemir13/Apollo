// Package totp generates time-based one-time passwords (RFC 6238) for services
// like Salesforce, and guards access behind a 4-digit PIN with an auto-lock
// window. Per-account secrets and the PIN hash live in the OS keychain (macOS
// Keychain / Linux Secret Service) — never in the plaintext config file.
package totp

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/bcrypt"
)

const pinKey = "mfa-pin"

// Service manages secrets, codes, the PIN and the lock state.
type Service struct {
	service        string     // keychain service namespace
	mu             sync.Mutex // locking the process for avoid interruprion
	failedAttempts int        // tracks sequential failures
	lockoutUntil   time.Time  // tracks when the user can try again
	unlockedUntil  time.Time
}

// New returns a Service that namespaces all keychain entries under `service`.
func New(service string) *Service {
	return &Service{service: service}
}

// normalizeSecret cleans a secret as copied from a setup screen (Salesforce
// shows it lower-cased in space-separated groups).
func normalizeSecret(secret string) string {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	secret = strings.ReplaceAll(secret, " ", "")
	secret = strings.ReplaceAll(secret, "-", "")
	return secret
}

// SetSecret validates a base32 TOTP secret and stores it in the keychain under
// the account id.
func (s *Service) SetSecret(id, secret string) error {
	secret = normalizeSecret(secret)
	if secret == "" {
		return fmt.Errorf("empty secret")
	}
	// Validate by attempting to generate a code now.
	if _, err := totp.GenerateCode(secret, time.Now()); err != nil {
		return fmt.Errorf("invalid authenticator key: %w", err)
	}
	return keyring.Set(s.service, "totp:"+id, secret)
}

// DeleteSecret removes an account's secret from the keychain.
func (s *Service) DeleteSecret(id string) error {
	err := keyring.Delete(s.service, "totp:"+id)
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}

// Code returns the current code and seconds remaining in its 30s window.
func (s *Service) Code(id string) (code string, secondsRemaining int, err error) {
	if !s.Unlocked() {
		return "", 0, fmt.Errorf("authenticator is locked")
	}

	secret, err := keyring.Get(s.service, "totp:"+id)
	if err != nil {
		return "", 0, err
	}
	now := time.Now()
	code, err = totp.GenerateCode(secret, now)
	if err != nil {
		return "", 0, err
	}
	secondsRemaining = 30 - int(now.Unix()%30)
	return code, secondsRemaining, nil
}

// ---- PIN -------------------------------------------------------------------

// SetPin stores a bcrypt hash of the PIN in the keychain.
func (s *Service) SetPin(pin string) error {
	pin = strings.TrimSpace(pin)
	if len(pin) < 4 {
		return fmt.Errorf("PIN must be at least 4 digits")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), 13)
	if err != nil {
		return err
	}
	return keyring.Set(s.service, pinKey, string(hash))
}

// HasPin reports whether a PIN has been configured.
func (s *Service) HasPin() bool {
	v, err := keyring.Get(s.service, pinKey)
	return err == nil && v != ""
}

// VerifyPin checks a PIN against the stored hash with built-in rate throttling.
func (s *Service) VerifyPin(pin string) bool {
	// mutex lock guarantees the each attempt will be verified individually to avoid
	// racing condition, and taking advantage of concurrency while validation is being conducted
	s.mu.Lock()
	// check if the service is currently under a hard lockout window
	if time.Now().Before(s.lockoutUntil) {
		s.mu.Unlock()
		slog.Warn("PIN verification rejected: service is temporarily locked out")
		return false
	}
	// unlock mutex to process the heavy load concurrently
	s.mu.Unlock()

	// fetch the hash from the OS Keychain
	hash, err := keyring.Get(s.service, pinKey)
	if err != nil {
		return false
	}

	// perform the cryptographic comparison
	isValid := bcrypt.CompareHashAndPassword([]byte(hash), []byte(strings.TrimSpace(pin))) == nil

	// lock again for verification and individually register each failed attempts
	s.mu.Lock()
	defer s.mu.Unlock()

	if isValid {
		// clear the slate completely on a successful validation
		s.failedAttempts = 0
		s.lockoutUntil = time.Time{}
		return true
	}

	// handle a failed verification attempt
	s.failedAttempts++

	// multi level lockout to secure the system
	if s.failedAttempts == 3 {
		// Tier 2: Hard Lockout. Trigger a mandatory 30 second cooldown period.
		s.lockoutUntil = time.Now().Add(30 * time.Second)
		slog.Warn("Too many failed PIN attempts. Locked out for 30 seconds.", "total_failures", s.failedAttempts)
	} else if s.failedAttempts > 3 {
		// Tier 3: Hard Lockout. Trigger a mandatory 5 minutes cooldown period.
		s.lockoutUntil = time.Now().Add(5 * time.Minute)
		slog.Warn("Too many failed PIN attempts. Locked out for 5 minutes.", "total_failures", s.failedAttempts)
	} else {
		// Tier 1: Progressive Delay. Intentionally sleep the goroutine to slow down scripts.
		// Attempt 1 fails = 1 second freeze. Attempt 2 fails = 2 second freeze.
		penaltyDelay := time.Duration(s.failedAttempts) * time.Second
		slog.Warn("Invalid PIN attempt. Introducing penalty delay.", "delay", penaltyDelay)

		s.mu.Unlock()
		time.Sleep(penaltyDelay)
		s.mu.Lock()
	}

	return false
}

// ClearPin removes the PIN (also locks).
func (s *Service) ClearPin() error {
	s.Lock()
	err := keyring.Delete(s.service, pinKey)
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}

// Unlock verifies the PIN and opens the access window for `window`.
func (s *Service) Unlock(pin string, window time.Duration) bool {
	if !s.VerifyPin(pin) {
		return false
	}
	s.mu.Lock()
	s.unlockedUntil = time.Now().Add(window)
	s.mu.Unlock()
	return true
}

// Lock closes the access window immediately.
func (s *Service) Lock() {
	s.mu.Lock()
	s.unlockedUntil = time.Time{}
	s.mu.Unlock()
}

// Unlocked reports whether the access window is currently open. A Service with
// no PIN is always considered locked (codes require a PIN to be set first).
func (s *Service) Unlocked() bool {
	if !s.HasPin() {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Now().Before(s.unlockedUntil)
}

// SecondsUntilLock returns how long the window stays open, 0 if locked.
func (s *Service) SecondsUntilLock() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := time.Until(s.unlockedUntil)
	if d <= 0 {
		return 0
	}
	return int(d.Seconds())
}
