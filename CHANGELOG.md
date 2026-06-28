# Changelog

## [1.0.0] — 2025-06-26

Initial public release.

### Features

- **Clock** — local time and date display
- **Weather** — current conditions via Open-Meteo (auto-detect location or manual city, °C/°F)
- **System** — CPU %, RAM usage, battery state, uptime, and top-5 processes by CPU
- **GitHub** — open pull requests across configured repos, filtered by login
- **Teams** — unread chat count and previews via Microsoft Graph (device-code OAuth) or macOS Notification Center (no sign-in required)
- **2FA / TOTP** — RFC 6238 codes for multiple accounts; secrets stored in the OS keychain; optional PIN with 5-minute auto-lock

### Platform support

| Platform | Architecture |
|---|---|
| macOS 10.13+ | Universal (Apple Silicon + Intel) |
| Ubuntu 22.04+ / Debian | x86-64 |
| Windows 10+ | x86-64 |

### Known limitations

- Teams Graph source requires an Azure AD app registration (see README)
- Teams local source (macOS) requires Full Disk Access in System Settings
- Linux requires a compositor for window transparency; headless environments show a solid background
- Battery stats return `n/a` on desktop machines and CI runners without a battery
