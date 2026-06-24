package totp

import (
	"strings"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

func TestSetPinTooLong(t *testing.T) {
	// bcrypt rejects passwords longer than 72 bytes — no keyring needed.
	if err := New("Apollo-Widget-unittest").SetPin(strings.Repeat("1", 100)); err == nil {
		t.Fatal("SetPin should error on overly long PIN")
	}
}

func TestCodeBadStoredSecret(t *testing.T) {
	const svc = "Apollo-Widget-unittest"
	if !keyringAvailable(t, svc) {
		t.Skip("OS keyring not available")
	}
	// Store an invalid secret directly, bypassing SetSecret's validation.
	if err := keyring.Set(svc, "totp:bad", "@@@not-base32@@@"); err != nil {
		t.Fatal(err)
	}
	defer keyring.Delete(svc, "totp:bad")
	if _, _, err := New(svc).Code("bad"); err == nil {
		t.Fatal("Code should error on an unparseable stored secret")
	}
}

func TestNormalizeSecret(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"jbswy3dpehpk3pxp", "JBSWY3DPEHPK3PXP"},
		{"jbsw y3dp ehpk 3pxp", "JBSWY3DPEHPK3PXP"},
		{"JBSW-Y3DP-EHPK", "JBSWY3DPEHPK"},
		{"  pad  ", "PAD"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := normalizeSecret(tt.in); got != tt.want {
				t.Errorf("normalizeSecret(%q)=%q want %q", tt.in, got, tt.want)
			}
		})
	}
}

// keyringAvailable reports whether the OS keyring works in this environment.
func keyringAvailable(t *testing.T, svc string) bool {
	t.Helper()
	if err := keyring.Set(svc, "probe", "x"); err != nil {
		return false
	}
	_ = keyring.Delete(svc, "probe")
	return true
}

func TestServiceFlow(t *testing.T) {
	const svc = "Apollo-Widget-unittest"
	if !keyringAvailable(t, svc) {
		t.Skip("OS keyring not available in this environment")
	}
	s := New(svc)
	const id = "acct1"
	// Clean any leftovers from a prior run, and clean up afterwards.
	cleanup := func() {
		_ = s.DeleteSecret(id)
		_ = s.ClearPin()
	}
	cleanup()
	t.Cleanup(cleanup)

	// --- PIN absent ---
	if s.HasPin() {
		t.Fatal("HasPin should be false initially")
	}
	if s.Unlocked() {
		t.Fatal("Unlocked should be false without a PIN")
	}
	if s.VerifyPin("0000") {
		t.Fatal("VerifyPin should be false without a PIN")
	}
	if err := s.ClearPin(); err != nil {
		t.Fatalf("ClearPin with no pin should be nil, got %v", err)
	}

	// --- PIN errors and set ---
	if err := s.SetPin("12"); err == nil {
		t.Fatal("SetPin should reject short PIN")
	}
	if err := s.SetPin("1234"); err != nil {
		t.Fatalf("SetPin: %v", err)
	}
	if !s.HasPin() {
		t.Fatal("HasPin should be true after SetPin")
	}
	if s.VerifyPin("9999") {
		t.Fatal("wrong PIN should not verify")
	}
	if !s.VerifyPin("1234") {
		t.Fatal("correct PIN should verify")
	}

	// --- lock window ---
	if s.Unlock("9999", time.Hour) {
		t.Fatal("Unlock with wrong PIN should fail")
	}
	if !s.Unlock("1234", time.Hour) {
		t.Fatal("Unlock with correct PIN should succeed")
	}
	if !s.Unlocked() {
		t.Fatal("should be unlocked")
	}
	if s.SecondsUntilLock() <= 0 {
		t.Fatal("SecondsUntilLock should be positive while unlocked")
	}
	s.Lock()
	if s.Unlocked() {
		t.Fatal("should be locked after Lock")
	}
	if s.SecondsUntilLock() != 0 {
		t.Fatal("SecondsUntilLock should be 0 when locked")
	}
	// zero-length window unlocks then immediately expires
	if !s.Unlock("1234", 0) {
		t.Fatal("Unlock should return true even with 0 window")
	}
	if s.Unlocked() {
		t.Fatal("0-window should already be expired")
	}

	// --- secrets ---
	if err := s.SetSecret(id, ""); err == nil {
		t.Fatal("empty secret should error")
	}
	if err := s.SetSecret(id, "!!!notbase32!!!"); err == nil {
		t.Fatal("invalid secret should error")
	}
	if _, _, err := s.Code("missing"); err == nil {
		t.Fatal("Code for missing secret should error")
	}
	if err := s.SetSecret(id, "JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("SetSecret valid: %v", err)
	}
	code, secs, err := s.Code(id)
	if err != nil {
		t.Fatalf("Code: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("code %q not 6 digits", code)
	}
	if secs < 1 || secs > 30 {
		t.Errorf("seconds %d out of range", secs)
	}
	if err := s.DeleteSecret(id); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}
	if err := s.DeleteSecret(id); err != nil {
		t.Fatalf("DeleteSecret on missing should be nil, got %v", err)
	}
}
