// Package teams reads unread Microsoft Teams chat messages via Microsoft Graph.
//
// Authentication uses the OAuth 2.0 device-code flow against an Azure AD app
// registration (a public client — no secret required). Tokens are cached on
// disk so the user only signs in once.
//
// Note on "favorites": Microsoft Graph does not expose the Teams "favorite"
// flag for chats. Code approximate the user's intent by surfacing every chat that
// has an unread message (a newer message than the last one the user read), and
// optionally filtering to a configured allow-list of chat names.
package teams

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

// scopes requested from Graph. offline_access/openid are added by MSAL.
var scopes = []string{"Chat.Read", "User.Read"}

// Service manages Graph access and token caching.
type Service struct {
	clientID string
	tenantID string
	cache    *fileCache
	client   *public.Client
}

// New builds a Service. clientID/tenantID come from config; cachePath is where
// the token cache is persisted. A zero clientID means Teams is not configured.
func New(clientID, tenantID, cachePath string) *Service {
	if tenantID == "" {
		tenantID = "common"
	}
	return &Service{
		clientID: clientID,
		tenantID: tenantID,
		cache:    &fileCache{path: cachePath},
	}
}

// Configured reports whether a client ID has been provided.
func (s *Service) Configured() bool { return strings.TrimSpace(s.clientID) != "" }

func (s *Service) ensureClient() error {
	if s.client != nil {
		return nil
	}
	if !s.Configured() {
		return fmt.Errorf("Teams is not configured (missing Azure client ID)")
	}
	authority := "https://login.microsoftonline.com/" + s.tenantID
	c, err := public.New(s.clientID,
		public.WithAuthority(authority),
		public.WithCache(s.cache),
	)
	if err != nil {
		return err
	}
	s.client = &c
	return nil
}

// silentToken tries to get a token from the cache without user interaction.
func (s *Service) silentToken(ctx context.Context) (string, error) {
	if err := s.ensureClient(); err != nil {
		return "", err
	}
	accounts, err := s.client.Accounts(ctx)
	if err != nil || len(accounts) == 0 {
		return "", fmt.Errorf("no cached account")
	}
	res, err := s.client.AcquireTokenSilent(ctx, scopes, public.WithSilentAccount(accounts[0]))
	if err != nil {
		return "", err
	}
	return res.AccessToken, nil
}

// Login runs the device-code flow. prompt is called once with the user code and
// URL to display; the call blocks until the user completes sign-in or ctx is
// cancelled.
func (s *Service) Login(ctx context.Context, prompt func(DeviceCode)) error {
	if err := s.ensureClient(); err != nil {
		return err
	}
	dc, err := s.client.AcquireTokenByDeviceCode(ctx, scopes)
	if err != nil {
		return err
	}
	prompt(DeviceCode{
		UserCode:        dc.Result.UserCode,
		VerificationURL: dc.Result.VerificationURL,
		Message:         dc.Result.Message,
	})
	_, err = dc.AuthenticationResult(ctx)
	return err
}

// LoggedIn reports whether a usable cached token exists.
func (s *Service) LoggedIn(ctx context.Context) bool {
	_, err := s.silentToken(ctx)
	return err == nil
}

// getTokenFn is indirected so tests can inject a fake token without a real MSAL flow.
var getTokenFn = func(s *Service, ctx context.Context) (string, error) {
	return s.silentToken(ctx)
}

// GetUnread fetches chats with unread messages. nameFilter, if non-empty, keeps
// only chats whose name contains one of the given substrings (case-insensitive)
// — used to emulate a "favorites" allow-list.
func (s *Service) GetUnread(ctx context.Context, nameFilter []string) (Result, error) {
	token, err := getTokenFn(s, ctx)
	if err != nil {
		return Result{NeedsLogin: true}, nil
	}

	chats, err := fetchChats(ctx, token)
	if err != nil {
		return Result{}, err
	}

	var out Result
	for _, c := range chats {
		if !c.unread() {
			continue
		}
		name := c.displayName()
		if len(nameFilter) > 0 && !matchesAny(name, nameFilter) {
			continue
		}
		out.UnreadChats = append(out.UnreadChats, Chat{
			ID:        c.ID,
			Name:      name,
			Preview:   c.preview(),
			From:      c.sender(),
			Timestamp: c.LastMessagePreview.CreatedDateTime,
		})
	}
	out.TotalUnread = len(out.UnreadChats)
	return out, nil
}

func matchesAny(name string, subs []string) bool {
	ln := strings.ToLower(name)
	for _, s := range subs {
		s = strings.TrimSpace(strings.ToLower(s))
		if s != "" && strings.Contains(ln, s) {
			return true
		}
	}
	return false
}

// parseTime is a tolerant RFC3339 parser used to compare read timestamps.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Graph sometimes returns fractional seconds with many digits.
		t, _ = time.Parse("2006-01-02T15:04:05.9999999Z", s)
	}
	return t
}
