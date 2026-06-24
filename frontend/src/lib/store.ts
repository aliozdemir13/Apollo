// Central reactive state: which view is showing, the data behind each view,
// auto-refresh cadence, and the actions wired to the device buttons.

import { writable, get, derived } from "svelte/store";
import {
  GetSettings,
  SaveSettings,
  GetWeather,
  GetSystemStats,
  GetTopProcesses,
  OpenURL,
  MFAGetStatus,
  MFAGetCodes,
  MFAUnlock,
  MFALock,
} from "../../wailsjs/go/main/App";
import { EventsOn, ClipboardSetText } from "../../wailsjs/runtime/runtime";

// ---- types ----------------------------------------------------------------

export interface Async<T> {
  loading: boolean;
  error?: string;
  data?: T;
}

export interface DeviceCode {
  userCode: string;
  verificationUrl: string;
  message: string;
}

const ALL_VIEWS = ["clock", "weather", "system", "totp"];

// Seconds between auto-refreshes per view. Clock updates from the 1s tick.
// totp is handled by a dedicated per-second tick (mfaTick), not this cadence.
const CADENCE: Record<string, number> = {
  clock: 0,
  weather: 600,
  system: 3,
  totp: 99999,
};

// ---- stores ---------------------------------------------------------------

export const views = writable<string[]>(ALL_VIEWS);
export const index = writable(0);
export const now = writable(new Date());

export const weather = writable<Async<any>>({ loading: true });
export const system = writable<Async<any>>({ loading: true });
// Inner sub-view of the system screen, toggled by the orange button.
export const systemMode = writable<"stats" | "apps">("stats");
export const topApps = writable<Async<any>>({ loading: true });

// ---- MFA / 2FA state ----
export const mfaHasPin = writable(false);
export const mfaAccountCount = writable(0);
export const mfaUnlocked = writable(false);
export const mfaCodes = writable<any[]>([]);
export const mfaPin = writable(""); // digits entered on the lock screen
export const mfaError = writable("");
let mfaLockAt = 0; // epoch ms when the unlock window expires
let lastCodeSecs = 0;

export const settings = writable<any>(null);
export const settingsOpen = writable(false);
export const deviceCode = writable<DeviceCode | null>(null);

export const currentView = derived([views, index], ([$views, $index]) =>
  $views.length ? $views[$index % $views.length] : "clock"
);

const lastFetched: Record<string, number> = {};

// ---- fetching --------------------------------------------------------------

async function load<T>(
  store: typeof weather,
  fn: () => Promise<T>,
  id: string
): Promise<void> {
  store.update((s) => ({ ...s, loading: true }));
  try {
    const data = await fn();
    store.set({ loading: false, data });
  } catch (e: any) {
    store.set({ loading: false, error: String(e?.message || e) });
  }
  lastFetched[id] = Date.now();
}

export function refresh(id: string, force = false): void {
  const due =
    force ||
    !lastFetched[id] ||
    (CADENCE[id] > 0 && Date.now() - lastFetched[id] > CADENCE[id] * 1000);
  if (!due) return;

  switch (id) {
    case "weather":
      load(weather, GetWeather, id);
      break;
    case "system":
      load(system, GetSystemStats, id);
      load(topApps, GetTopProcesses, "topApps");
      break;
    case "totp":
      refreshMFA();
      break;
  }
}

// ---- in-app screen navigation (horizontal left/right buttons) --------------

// screenNext / screenPrev switch between sub-screens of the current view.
// Views without sub-screens ignore them.
export function screenNext(): void {
  changeScreen(1);
}
export function screenPrev(): void {
  changeScreen(-1);
}
function changeScreen(dir: number): void {
  const v = get(currentView);
  if (v === "system") {
    systemMode.update((m) => (m === "stats" ? "apps" : "stats"));
  }
}

// hasSubScreens reports whether the current view responds to left/right.
export function hasSubScreens(view: string): boolean {
  return view === "system";
}

// ---- MFA actions -----------------------------------------------------------

// refreshMFA syncs lock/account status from the backend and, if unlocked,
// loads the current codes.
export async function refreshMFA(): Promise<void> {
  const st = await MFAGetStatus();
  mfaHasPin.set(st.hasPin);
  mfaAccountCount.set(st.accountCount);
  mfaUnlocked.set(st.unlocked);
  if (st.unlocked) {
    mfaLockAt = Date.now() + st.secondsUntilLock * 1000;
    await fetchCodes();
  } else {
    mfaCodes.set([]);
  }
}

async function fetchCodes(): Promise<void> {
  // Never request codes while the screen is locked.
  if (!get(mfaUnlocked)) {
    mfaCodes.set([]);
    return;
  }
  const res = await MFAGetCodes();
  if (res.locked) {
    mfaUnlocked.set(false);
    mfaCodes.set([]);
    return;
  }
  mfaCodes.set(res.entries || []);
  mfaLockAt = Date.now() + res.secondsUntilLock * 1000;
}

// pinPush appends a digit to the lock-screen entry; at 4 digits it auto-submits.
export function pinPush(d: string): void {
  if (get(mfaUnlocked)) return;
  const cur = get(mfaPin);
  if (cur.length >= 4) return;
  const next = cur + d;
  mfaPin.set(next);
  if (next.length === 4) unlockWithPin();
}

export function pinBackspace(): void {
  mfaPin.update((p) => p.slice(0, -1));
  mfaError.set("");
}

export async function unlockWithPin(): Promise<void> {
  const pin = get(mfaPin);
  const ok = await MFAUnlock(pin);
  mfaPin.set("");
  if (ok) {
    mfaError.set("");
    await refreshMFA();
  } else {
    mfaError.set("WRONG PIN");
  }
}

export async function lockMFA(): Promise<void> {
  await MFALock();
  mfaUnlocked.set(false);
  mfaCodes.set([]);
  mfaPin.set("");
}

export function copyCode(code: string): void {
  if (code) ClipboardSetText(code);
}

// mfaTick runs every second while on the 2FA view: it auto-locks when the
// window expires and re-fetches codes when the 30s TOTP window rolls over —
// all without per-second keychain reads.
function mfaTick(): void {
  if (get(currentView) !== "totp" || !get(mfaUnlocked)) return;
  if (Date.now() >= mfaLockAt) {
    lockMFA();
    return;
  }
  const secs = 30 - (Math.floor(Date.now() / 1000) % 30);
  if (secs > lastCodeSecs) {
    fetchCodes(); // window wrapped — pull fresh codes
  }
  lastCodeSecs = secs;
}

// ---- navigation (device buttons) ------------------------------------------

export function next(): void {
  const n = get(views).length;
  if (!n) return;
  index.set((get(index) + 1) % n);
  refresh(get(currentView));
}

export function prev(): void {
  const n = get(views).length;
  if (!n) return;
  index.set((get(index) - 1 + n) % n);
  refresh(get(currentView));
}

// The orange check button: context action. Refreshes the current view, 
export async function confirm(): Promise<void> {
  const view = get(currentView);
  // 2FA: the orange button is a lock (not refresh). When unlocked it locks the
  // screen immediately; the lock screen handles its own PIN entry.
  if (view === "totp") {
    if (get(mfaUnlocked)) lockMFA();
    return;
  }
  // Every other view: the orange button refreshes the current screen.
  refresh(view, true);
}

export function openExternal(url: string): void {
  OpenURL(url);
}

// ---- settings --------------------------------------------------------------

export async function loadSettings(): Promise<void> {
  const s = await GetSettings();
  settings.set(s);
  if (s?.views?.length) views.set(s.views);
}

export async function saveSettings(s: any): Promise<void> {
  await SaveSettings(s);
  settings.set(s);
  if (s?.views?.length) {
    views.set(s.views);
    index.set(0);
  }
  // Force re-fetch of data that may have changed.
  lastFetched["weather"] = 0;
  refresh(get(currentView), true);
}

// ---- lifecycle -------------------------------------------------------------

let timer: number | undefined;

export async function start(): Promise<void> {

  await loadSettings();
  refresh(get(currentView), true);

  timer = window.setInterval(() => {
    now.set(new Date());
    // Refresh the active view if its cadence has elapsed.
    refresh(get(currentView));
    // 2FA countdown / auto-lock / code rollover.
    mfaTick();
  }, 1000);
}

export function stop(): void {
  if (timer) window.clearInterval(timer);
}
