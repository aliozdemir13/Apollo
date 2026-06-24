# Retro Widget

A desktop widget styled like a retro LCD gadget. It floats on top of your other
windows and cycles through several **views** using its physical-looking buttons:

| View | Shows | Data source |
|------|-------|-------------|
| **Clock** | Time + date (with a mini weather line) | local clock |
| **Weather** | Condition, temperature, feels-like, humidity, wind | [Open-Meteo](https://open-meteo.com) (no API key) |
| **System** | CPU %, RAM, battery, uptime — **✓ toggles a top-5 apps list** | local (`/proc`, `top`, `ps`, IOKit) |
| **2FA** | Rotating TOTP codes for one or more Salesforce orgs, PIN-locked | local (RFC 6238); secrets in OS keychain |

Built with **[Wails v2](https://wails.io)** — a Go backend compiled together with
a Svelte frontend into a single native binary. Runs on **macOS** and **Ubuntu/Linux**.

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

The **horizontal ‹ ›** buttons switch between a view's sub-screens (they're dimmed
on views that have none):

- **System** — stats gauges ↔ **top-5 apps**.

The orange button **refreshes** the current screen on every view **except 2FA**,
where it is a **padlock** that locks the screen:

- **2FA** — tap a tile to copy its code; the orange padlock locks the screen
  (codes aren't fetched at all while locked). On the lock screen, type your PIN.

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
Name=Retro Widget
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

---

## Configuration

Open **Settings** in the widget (press <kbd>S</kbd> or the ⚙ icon). Everything is
stored in a single JSON file:

- **macOS**: `~/Library/Application Support/Apollo-Widget/config.json`
- **Linux**: `~/.config/Apollo-Widget/config.json`

You can edit it by hand too; the path is shown at the bottom of the settings screen.

### Weather

Leave **Location** blank to auto-detect by IP, or type a city (e.g. `Munich`).
Choose Celsius or Fahrenheit. Coordinates are resolved once and cached.

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

## Project layout

```
.
├── main.go                 # Wails app + window options (frameless, always-on-top)
├── app.go                  # the single struct bound to the frontend
├── internal/
│   ├── config/             # JSON settings load/save (+ view migration)
│   ├── weather/            # Open-Meteo client + geocoding + IP detection
│   ├── sysstats/           # CPU / RAM / battery / uptime + top processes (per-OS, no cgo for CPU)
│   └── totp/               # RFC 6238 codes, OS-keychain secrets, PIN + auto-lock
└── frontend/
    └── src/
        ├── App.svelte      # the device chrome (screen, rocker, ✓ button)
        ├── lib/store.ts    # view state, polling, button actions
        ├── views/          # Clock, Weather, System, Totp
        ├── components/     # DotMatrix glyph
        └── Settings.svelte # settings overlay
```

### Implementation note

On Apple Silicon, `gopsutil`'s `cpu`/`host` packages transitively import
`github.com/shoenig/go-m1cpu`, whose cgo `init()` segfaults on recent macOS. This
project therefore avoids those packages and computes CPU% and uptime with small
per-OS helpers (`internal/sysstats/cpu_linux.go`, `cpu_darwin.go`). Only
`gopsutil/mem` and `distatus/battery` are used directly.
