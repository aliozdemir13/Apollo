package teams

import (
	"context"
	"os"
	"sync"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

// fileCache persists the MSAL token cache to a single file on disk so the user
// stays signed in across restarts. It implements cache.ExportReplace.
type fileCache struct {
	path string
	mu   sync.Mutex
}

// Replace loads the cached tokens into MSAL on startup / before a token op.
func (f *fileCache) Replace(ctx context.Context, c cache.Unmarshaler, hints cache.ReplaceHints) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, err := os.ReadFile(f.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return c.Unmarshal(data)
}

// Export writes MSAL's token cache back to disk after a token op.
func (f *fileCache) Export(ctx context.Context, c cache.Marshaler, hints cache.ExportHints) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, err := c.Marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(f.path, data, 0o600)
}
