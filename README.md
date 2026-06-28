# Apollo Widget

A desktop widget styled like a retro LCD gadget. It floats on top of your other
windows and cycles through several **views** using its physical-looking buttons:

| View | Shows | Data source |
|------|-------|-------------|
| **Clock** | Time + date (with a mini weather line) | local clock |
| **Weather** | Condition, temperature, feels-like, humidity, wind | [Open-Meteo](https://open-meteo.com) (no API key) |
| **System** | CPU %, RAM, battery, uptime — **✓ toggles a top-5 apps list** | local (`/proc`, `top`, `ps`, IOKit) |
| **GitHub** | Open PRs, PRs awaiting your review, latest CI runs, and per-repo errors (✓ cycles) | GitHub REST API (personal access token) |
| **Teams** | Unread chat messages (optionally filtered to "favorites") | Microsoft Graph (OAuth) **or** local macOS notifications (no keys) |
| **2FA** | Rotating TOTP codes for one or more Salesforce orgs, PIN-locked | local (RFC 6238); secrets in OS keychain |

Built with **[Wails v2](https://wails.io)** — a Go backend compiled together with
a Svelte frontend into a single native binary. Runs on **macOS** and **Ubuntu/Linux**.

**Important**: This is a learning project for Svelte, OS-level processes and build scripts. Portions of the Svelte frontend, OS-level process parsing and build scripts were developed with **AI assistance**. **All code has been reviewed, tested and is understood by the maintainer. 

---

## Controls

The widget is driven entirely by the on-device buttons (or the keyboard):

| Action | Button | Key |
|--------|--------|-----|
| Switch **app** (view) | ▲ / ▼ vertical rocker | <kbd>↑</kbd> / <kbd>↓</kbd> |
| Switch **in-app screen** | ‹ / › horizontal rocker | <kbd>←</kbd> / <kbd>→</kbd> |
| Refresh current screen | ⟳ (orange) | <kbd>Enter</kbd> |
| Open settings | hover top-left ⚙ | <kbd>S</kbd> |
| Quit | hover top-left ⏻ | — |

Drag the gray body to move the window. On the GitHub view, click a PR to open it
in your browser.

The **horizontal ‹ ›** buttons switch between a view's sub-screens (they're dimmed
on views that have none):

- **System** — stats gauges ↔ **top-5 apps**.
- **GitHub** — **Open PRs → To review → Workflows → Errors**.

The orange button **refreshes** the current screen on every view **except 2FA**,
where it is a **padlock** that locks the screen:

- **2FA** — tap a tile to copy its code; the orange padlock locks the screen
  (codes aren't fetched at all while locked). On the lock screen, type your PIN.
- **Teams** — when signed out, the orange button starts sign-in instead of refreshing.

> New views released in an update surface automatically the first time you launch
> the new build; you can then disable any you don't want in Settings → Views.

---

## Prerequisites

- **Go** ≥ 1.25 (see `go.mod`)
- **[Bun](https://bun.sh)** (the frontend package manager — this project is configured to use it)
- **[Wails CLI](https://wails.io)** v2: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Platform toolchains:
  - **macOS**: Xcode command-line tools (`xcode-select --install`)
  - **Ubuntu/Debian**: `sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev`
    (on older releases the package is `libwebkit2gtk-4.0-dev`)
  - **2FA keychain on Linux**: the TOTP secrets use the Secret Service API, so a
    keyring daemon must be running — `sudo apt install gnome-keyring` (or KWallet).
    macOS uses the built-in Keychain; nothing extra needed.

Verify your environment with `wails doctor`.

---

## Development

Hot-reloads the frontend and rebuilds Go on change:

```sh
wails dev
```

## Build a distributable app

```sh
wails build
```

Output:

- **macOS** → `build/bin/Apollo-Widget.app`
- **Linux** → `build/bin/Apollo-Widget` (a single binary)
- **Windows** → `build/bin/Apollo-Widget.exe` (a single binary)

> Cross-compiling between macOS and Linux is **not** supported because each links
> against the native system webview (WebKit / WebKitGTK). Build **on** the target
> OS. See the Ubuntu walkthrough below.

---

## Building on Ubuntu

Wails supports Linux fully — the widget runs natively via WebKitGTK. You just have
to compile **on** Ubuntu (a real machine, VM, or WSL2 with a GUI), not on macOS.
The repo already handles the Linux specifics (no code changes needed): the macOS
transparency hook compiles to a no-op, CPU/uptime come from `/proc`, and the 2FA
secrets use the Secret Service instead of the macOS Keychain.

Tested on **Ubuntu 22.04 / 24.04**.

### 1. System dependencies

```sh
sudo apt update
sudo apt install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev gnome-keyring
```

- `libwebkit2gtk-4.1-dev` is on 22.04+. On older releases install
  `libwebkit2gtk-4.0-dev` instead (and drop the `-tags webkit2_41` flag below).
- `gnome-keyring` provides the Secret Service the **2FA** view needs to store TOTP
  secrets. (KWallet works too on KDE.)

### 2. Go ≥ 1.25

Ubuntu's apt Go is usually too old. Install the official toolchain:

```sh
curl -fsSL https://go.dev/dl/go1.25.0.linux-amd64.tar.gz | sudo tar -C /usr/local -xz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc
go version   # → go1.25.x
```

### 3. Bun + Wails CLI

```sh
curl -fsSL https://bun.sh/install | bash          # frontend package manager
source ~/.bashrc
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails doctor                                       # should report all green
```

### 4. Build

From the project root:

```sh
# Ubuntu 22.04+/24.04 (WebKitGTK 4.1):
wails build -tags webkit2_41

# Older Ubuntu (WebKitGTK 4.0):
wails build
```

Output: a single binary at **`build/bin/Apollo-Widget`**. Run it with `./build/bin/Apollo-Widget`.

> `wails dev -tags webkit2_41` works the same way for live-reload development.

### 5. Run on login (optional)

Create `~/.config/autostart/Apollo-Widget.desktop` (adjust the path):

```ini
[Desktop Entry]
Type=Application
Name=Apollo-Widget
Exec=/home/YOU/Apollo-Widget/build/bin/Apollo-Widget
X-GNOME-Autostart-enabled=true
```

### Linux runtime notes

- **Transparency** requires a **compositing** window manager (GNOME/Mutter, KDE,
  and most Wayland sessions composite by default). Under a bare X11 WM with no
  compositor, the window corners may render black instead of transparent — start a
  compositor (e.g. `picom`) to fix it.
- **Always-on-top / frameless dragging** are honored by the WM; behavior varies
  slightly between GNOME, KDE, and tiling WMs.
- **2FA on a headless/SSH session** won't work — the Secret Service needs an active
  desktop login session with the keyring unlocked.

### macOS, for completeness

```sh
wails build      # → build/bin/Apollo-Widget.app
```

Add it via System Settings → General → Login Items to run on login.

## Building on Windows

---

## Configuration

Open **Settings** in the widget (press <kbd>S</kbd> or the ⚙ icon). Everything is
stored in a single JSON file:

- **macOS**: `~/Library/Application Support/Apollo-Widget/config.json`
- **Linux**: `~/.config/Apollo-Widget/config.json`
- **Windows**: `%AppData%\Apollo-Widget\config.json`

You can edit it by hand too; the path is shown at the bottom of the settings screen.

### Weather

Leave **Location** blank to auto-detect by IP, or type a city (e.g. `Berlin`).
Choose Celsius or Fahrenheit. Coordinates are resolved once and cached.

### GitHub PRs

1. Create a **fine-grained or classic personal access token** at
   <https://github.com/settings/tokens> with **read access to pull requests**
   (classic: `repo` scope). For the **Workflows** screen also grant
   **Actions: Read-only** (fine-grained); the classic `repo` scope already covers it.
2. Paste it into Settings → GitHub → **Token**.
3. List the repos to watch, one per line, as `owner/name`.
4. *(Optional)* Set **"Only my PRs"** to your GitHub login to filter the Open PRs
   screen to PRs you authored.

The GitHub view has four screens (press **✓** to cycle):

- **Open PRs** — open pull requests across your repos (optionally filtered to yours).
- **To review** — open PRs where you're a requested reviewer (`review-requested:@me`).
- **Workflows** — the latest GitHub Actions run per repo, with status (OK / FAIL / running).
- **Errors** — any per-repo fetch failures (bad token, no access, repo not found).

### Microsoft Teams

The Teams view has two **sources** (Settings → Teams → *Source*):

#### Local — macOS notifications (no keys)

> [!IMPORTANT]
> This feature only works on macOS!

If you can't register an Azure app, choose **Local**. It reads delivered Teams
notifications straight from the macOS Notification Center database — showing each
recent message's sender + preview, with **no API keys or infrastructure**.

- **Requires Full Disk Access**: System Settings → Privacy & Security → **Full Disk
  Access** → enable **Apollo-Widget** (during `wails dev`, grant it to your terminal
  instead). Until granted, the view shows "grant Full Disk Access".
- Shows **delivered notifications** (what Teams notified you about and you haven't
  cleared) — a proxy for recent chatter, not a true unread count.
- **macOS only.** On Linux there's no equivalent persisted store, so use Graph there.
- The **Favorites filter** still applies (limits to matching chat/sender names).

#### Microsoft Graph (Azure app)

For a true unread view, register a (free) Azure AD app once. It's a **public
client** — no secret is stored.

1. Go to the [Azure Portal → App registrations](https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade) → **New registration**.
2. Name it anything (e.g. `Apollo-Widget`). Under **Supported account types**, pick
   the option matching your org (usually *Accounts in this organizational directory only*).
3. After creating it, open **Authentication** → **Add a platform** → **Mobile and
   desktop applications**, and enable **"Allow public client flows"**
   (Authentication → *Advanced settings* → set **Allow public client flows** to **Yes**).
4. Under **API permissions**, add **Microsoft Graph → Delegated**: `Chat.Read` and
   `User.Read`. Grant admin consent if your tenant requires it.
5. Copy the **Application (client) ID** and **Directory (tenant) ID** from the app's
   Overview page into Settings → Teams.
   - Tenant can also be `common` or `organizations` if you prefer.
6. Switch to the **Teams** view in the widget and press **✓** to sign in. A code +
   URL appear on screen — open the URL, enter the code, and approve. The token is
   cached at `…/Apollo-Widget/teams_token.json` so you stay signed in.

**About "favorites":** Microsoft Graph does not expose the Teams *favorite* flag
for chats, so the widget shows every chat with an **unread** message. To emulate a
favorites list, add name substrings under **Favorites filter** (one per line) — only
matching chats will be shown.

### Salesforce 2FA (TOTP)

The **2FA** view acts as a standard TOTP authenticator (RFC 6238) — the same
category as Google Authenticator — so it can replace the mobile authenticator on
the Salesforce login screen. It supports multiple orgs, each with its own code.

> It does **not** replace the push-based *Salesforce Authenticator* app (approve/deny
> notifications); that uses an undocumented proprietary protocol. This is the
> "Use an authenticator app" / one-time-password method.

**Setup:**

1. In the widget: Settings → **2FA — Salesforce orgs** → set a 4-digit **unlock PIN**.
2. In Salesforce: your account's MFA setup → *"Connect an Authenticator App"* →
   click **"Can't scan the QR code?"** to reveal the **secret key** (a base32 string).
3. Back in Settings, enter an **Org name** and paste the **key**, then **+ Add org**.
   (Spaces and letter-case in the key don't matter.)
4. Salesforce will ask you to **confirm with the current 6-digit code** to finish
   enrollment — switch to the 2FA view, unlock with your PIN, and read it off.
5. From then on, the view shows each org's live code with a 30-second countdown.
   Click a row (or press ✓ for the top org) to **copy** the code.

**Security model — read this:**

- **Secrets live in the OS keychain** (macOS Keychain / Linux Secret Service), keyed
  by account ID. They are never written to `config.json`.
- The **PIN** is bcrypt-hashed in the keychain and gates the screen. The 2FA screen
  **auto-locks 5 minutes** after unlocking (enforced in the Go backend — codes can't
  be read while locked), then requires the PIN again.
- The PIN is a *local convenience lock*, not a cryptographic barrier against someone
  with full access to your machine.
- Putting a second factor on the same machine you log in from is inherently weaker
  than a separate phone. This is a convenience/security tradeoff — use it knowingly,
  and check that your org permits authenticator-app TOTP (most do unless they mandate
  Salesforce Authenticator specifically).

---

## Testing

The Go backend has table-driven unit tests. Run them with:

```sh
go test ./...                          # all tests
go test ./internal/... -cover          # per-package coverage
go test ./internal/... -coverpkg=./internal/... -coverprofile=cov.out && go tool cover -html=cov.out
```

Coverage of the `internal/` packages is **~90%**, with several at 100%:

| Package | Coverage | Notes |
|---------|----------|-------|
| `weather` | 100% | HTTP paths via `httptest` |
| `github` | 100% | HTTP paths via `httptest` |
| `totp` | 100% | uses the OS keyring (test skips if unavailable) |
| `config` | 98% | 1 unreachable defensive branch (`json.Marshal` can't fail here) |
| `sysstats` | ~95% | parsing fully tested; remainder is the per-OS `ps` branch + battery hardware states |
| `teams` | ~75% | pure logic, the Graph client, the local-notification parser and the file cache are covered |

Some paths are **intentionally not unit-tested** because they require live external
state that can't be faithfully faked in `go test`:

- **MSAL device-code sign-in** (`teams.Login`) and the authenticated Graph fetch —
  need a real Azure app + interactive login.
- **The macOS notification DB read** (`readTeamsNotifications`) — needs the
  TCC-protected database + Full Disk Access.
- **The Wails-bound layer** (`main.go`, `app.go`, `chrome_darwin.go`) — needs the
  desktop runtime/window.

To keep the rest testable, HTTP endpoints are overridable package vars, the
notification reader and shell commands are indirected behind function vars, and the
parsing logic is split out from the I/O.

---

## Project layout

```
.
├── main.go                 # Wails app + window options (frameless, always-on-top)
├── app.go                  # the single struct bound to the frontend
├── internal/
│   ├── config/             # JSON settings load/save (+ view migration)
│   ├── weather/            # Open-Meteo client + geocoding + IP detection
│   ├── sysstats/           # CPU / RAM / battery / uptime + top processes (per-OS, no cgo for CPU)
│   ├── github/             # open PRs from selected repos
│   ├── teams/              # MS Graph + MSAL device-code auth + token cache
│   └── totp/               # RFC 6238 codes, OS-keychain secrets, PIN + auto-lock
└── frontend/
    └── src/
        ├── App.svelte      # the device chrome (screen, rocker, ✓ button)
        ├── lib/store.ts    # view state, polling, button actions
        ├── views/          # Clock, Weather, System, Github, Teams, Totp
        ├── components/     # DotMatrix glyph
        └── Settings.svelte # settings overlay
```

### Implementation note

On Apple Silicon, `gopsutil`'s `cpu`/`host` packages transitively import
`github.com/shoenig/go-m1cpu`, whose cgo `init()` segfaults on recent macOS. This
project therefore avoids those packages and computes CPU% and uptime with small
per-OS helpers (`internal/sysstats/cpu_linux.go`, `cpu_darwin.go`). Only
`gopsutil/mem` and `distatus/battery` are used directly.
