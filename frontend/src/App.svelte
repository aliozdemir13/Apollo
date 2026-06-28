<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import {
    start,
    stop,
    next,
    prev,
    confirm,
    screenNext,
    screenPrev,
    hasSubScreens,
    currentView,
    settingsOpen,
    teamsLoggedIn,
    deviceCode,
    mfaUnlocked,
    mfaHasPin,
    weather,
    settings,
  } from "./lib/store";
  import { Quit } from "../wailsjs/runtime/runtime";
  import DotMatrix from "./components/DotMatrix.svelte";
  import Clock from "./views/Clock.svelte";
  import Weather from "./views/Weather.svelte";
  import System from "./views/System.svelte";
  import Github from "./views/Github.svelte";
  import Teams from "./views/Teams.svelte";
  import Totp from "./views/Totp.svelte";
  import Settings from "./Settings.svelte";

  const VIEW_COMPONENT: Record<string, any> = {
    clock: Clock,
    weather: Weather,
    system: System,
    github: Github,
    teams: Teams,
    totp: Totp,
  };
  const MATRIX_KIND: Record<string, any> = {
    clock: "grid",
    weather: "sun",
    system: "cpu",
    github: "git",
    teams: "chat",
    totp: "lock",
  };

  // Map the live weather (WMO code + wind) to a dot-matrix glyph.
  function weatherGlyph(w: any): string {
    if (!w) return "sun";
    const c = w.code;
    if ([95, 96, 99].includes(c)) return "storm";
    if ([71, 73, 75, 77, 85, 86].includes(c)) return "snow";
    if ([51, 53, 55, 56, 57, 61, 63, 65, 66, 67, 80, 81, 82].includes(c)) return "rain";
    if (w.windSpeed >= 30 && [0, 1, 2, 3, 45, 48].includes(c)) return "wind";
    if ([2, 3, 45, 48].includes(c)) return "cloud";
    return "sun"; // 0/1 = clear / mainly clear
  }

  // Apply the chosen colour theme to the document root so the CSS variable
  // overrides in style.css take effect. Falls back to grey until settings load.
  $: if (typeof document !== "undefined") {
    document.documentElement.dataset.theme = $settings?.theme || "grey";
  }

  $: view = $currentView;
  $: matrixKind = view === "weather" ? weatherGlyph($weather.data) : MATRIX_KIND[view] || "grid";
  $: subScreens = hasSubScreens(view);

  // Contextual footer hint.
  $: hint =
    view === "teams" && $deviceCode
      ? "complete sign-in in browser"
      : view === "teams" && !$teamsLoggedIn
      ? "to connect"
      : view === "totp"
      ? !$mfaHasPin
        ? "set a PIN in settings"
        : $mfaUnlocked
        ? "tap a tile to copy · 🔒 locks"
        : "enter PIN to unlock"
      : "to refresh";

  function onKey(e: KeyboardEvent) {
    if ($settingsOpen) return;
    switch (e.key) {
      case "ArrowDown":
        next();
        break;
      case "ArrowUp":
        prev();
        break;
      case "ArrowRight":
        screenNext();
        break;
      case "ArrowLeft":
        screenPrev();
        break;
      case "Enter":
      case " ":
        confirm();
        break;
      case "s":
      case "S":
        settingsOpen.set(true);
        break;
    }
  }

  onMount(() => {
    start();
    window.addEventListener("keydown", onKey);
  });
  onDestroy(() => {
    stop();
    window.removeEventListener("keydown", onKey);
  });
</script>

<div class="stage">
  <div class="device">
    <div class="layout">
      <!-- LCD SCREEN -->
      <div class="screen">
        <div class="screen-top">
          <DotMatrix kind={matrixKind} />
        </div>

        <div class="screen-body">
          {#if !$settingsOpen}
            <svelte:component this={VIEW_COMPONENT[view] || Clock} />
          {/if}
        </div>

        <div class="screen-foot">
          <div class="pill lcd-mono">
            {#if view === "totp"}
              {hint}
            {:else}
              press <span class="ck">⟳</span> {hint}
            {/if}
          </div>
        </div>
      </div>

      <!-- CONTROL COLUMN -->
      <div class="controls">
        <div class="checker" aria-hidden="true">
          {#each Array(9) as _, i}
            <span class:on={[0, 2, 4, 6, 8].includes(i)}></span>
          {/each}
        </div>

        <!-- vertical rocker: switch apps (views) -->
        <div class="rocker">
          <button class="rk up" title="Previous app (↑)" on:click={prev}>
            <svg viewBox="0 0 24 24"><path d="M5 15l7-7 7 7" /></svg>
          </button>
          <div class="rk-divider"></div>
          <button class="rk down" title="Next app (↓)" on:click={next}>
            <svg viewBox="0 0 24 24"><path d="M5 9l7 7 7-7" /></svg>
          </button>
        </div>

        <!-- horizontal rocker: switch in-app screens -->
        <div class="hrocker" class:dim={!subScreens}>
          <button class="hk" title="Previous screen (←)" on:click={screenPrev} disabled={!subScreens}>
            <svg viewBox="0 0 24 24"><path d="M15 6 L9 12 L15 18" /></svg>
          </button>
          <div class="hk-divider"></div>
          <button class="hk" title="Next screen (→)" on:click={screenNext} disabled={!subScreens}>
            <svg viewBox="0 0 24 24"><path d="M9 6 L15 12 L9 18" /></svg>
          </button>
        </div>

        <button
          class="check"
          title={view === "totp" ? "Lock 2FA screen" : "Refresh (Enter)"}
          on:click={confirm}
        >
          {#if view === "totp"}
            <!-- padlock: this button locks the 2FA screen -->
            <svg viewBox="0 0 24 24" class="lock">
              <rect x="5" y="11" width="14" height="9" rx="2" />
              <path d="M8 11V8a4 4 0 0 1 8 0v3" />
            </svg>
          {:else}
            <!-- refresh: this button reloads the current screen (centered refresh-cw) -->
            <svg viewBox="0 0 24 24" class="refresh">
              <polyline points="23 4 23 10 17 10" />
              <polyline points="1 20 1 14 7 14" />
              <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
            </svg>
          {/if}
        </button>

        <div class="cbottom">
          <div class="toolrow">
            <button class="dbtn settings" title="Settings (s)" on:click={() => settingsOpen.set(true)}>
              <svg viewBox="0 0 24 24">
                <circle cx="12" cy="12" r="3" />
                <path
                  d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"
                />
              </svg>
            </button>
            <button class="dbtn power" title="Quit" on:click={() => Quit()}>
              <span class="led"></span>
              <svg viewBox="0 0 24 24">
                <path d="M18.36 6.64a9 9 0 1 1-12.73 0" />
                <line x1="12" y1="2" x2="12" y2="12" />
              </svg>
            </button>
          </div>
          <div class="master lcd-mono">Apollo</div>
        </div>
      </div>
    </div>

    {#if $settingsOpen}
      <Settings />
    {/if}
  </div>
</div>