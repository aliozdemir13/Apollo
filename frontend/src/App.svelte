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

<style>
  .stage {
    width: 100vw;
    height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  /* The gadget body */
  .device {
    position: relative;
    width: 340px;
    height: 372px;
    border-radius: var(--device-radius);
    padding: 14px;
    background: linear-gradient(160deg, var(--body-hi) 0%, var(--body-lo) 100%);
    box-shadow:
      inset 0 1px 0 rgba(255, 255, 255, 0.5),
      inset 0 -2px 6px rgba(0, 0, 0, 0.25),
      0 18px 40px rgba(0, 0, 0, 0.55),
      0 2px 4px rgba(0, 0, 0, 0.4);
    border: 1px solid var(--body-edge);
    --wails-draggable: drag;
  }

  /* bottom-right group: handle + settings/power buttons */
  .cbottom {
    margin-top: auto;
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
  }
  /* match the 56px width of the rocker / orange button so the buttons line up */
  .toolrow {
    width: 56px;
    display: flex;
    justify-content: space-between;
  }
  .dbtn {
    position: relative;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: linear-gradient(180deg, var(--btn-face-hi), var(--btn-face-lo));
    box-shadow:
      inset 0 1px 0 rgba(255, 255, 255, 0.85),
      0 1px 2px rgba(0, 0, 0, 0.35);
    display: flex;
    align-items: center;
    justify-content: center;
    --wails-draggable: no-drag;
  }
  .dbtn:active {
    transform: translateY(0.5px);
  }
  .dbtn svg {
    width: 13px;
    height: 13px;
    fill: none;
    stroke: #3a3833;
    stroke-width: 1.7;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  /* power button: looks like an orange light shining while the app is on */
  .dbtn.power {
    background:
      radial-gradient(circle at 50% 42%, rgba(224, 113, 47, 0.5), rgba(224, 113, 47, 0) 70%),
      linear-gradient(180deg, var(--btn-face-hi), var(--btn-face-lo));
    animation: powerGlow 2.6s ease-in-out infinite;
  }
  .dbtn.power svg {
    stroke: var(--accent);
    position: relative;
    z-index: 1;
  }
  .dbtn.power .led {
    position: absolute;
    inset: 0;
    border-radius: 50%;
    pointer-events: none;
  }
  @keyframes powerGlow {
    0%,
    100% {
      box-shadow:
        inset 0 1px 0 rgba(255, 255, 255, 0.85),
        0 1px 2px rgba(0, 0, 0, 0.35),
        0 0 5px 0.5px rgba(224, 113, 47, 0.45);
    }
    50% {
      box-shadow:
        inset 0 1px 0 rgba(255, 255, 255, 0.85),
        0 1px 2px rgba(0, 0, 0, 0.35),
        0 0 10px 2px rgba(224, 113, 47, 0.85);
    }
  }

  .layout {
    height: 100%;
    display: flex;
    gap: 12px;
  }

  /* LCD */
  .screen {
    flex: 1;
    background: radial-gradient(120% 120% at 30% 20%, var(--lcd-glow), var(--lcd-bg) 70%);
    border-radius: 12px;
    box-shadow:
      inset 0 2px 8px rgba(0, 0, 0, 0.9),
      inset 0 0 0 1px rgba(0, 0, 0, 0.6);
    padding: 14px 12px 12px;
    display: flex;
    flex-direction: column;
    --wails-draggable: no-drag;
    overflow: hidden;
  }
  .screen-top {
    display: flex;
    justify-content: center;
    padding-top: 2px;
  }
  .screen-body {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
  }
  .screen-foot {
    display: flex;
    justify-content: center;
    padding-bottom: 2px;
  }
  .pill {
    font-size: 10px;
    letter-spacing: 0.12em;
    color: var(--lcd-text);
    border: 1px solid rgba(236, 233, 223, 0.5);
    border-radius: 12px;
    padding: 4px 14px;
  }
  .pill .ck {
    color: var(--accent);
  }

  /* Controls column */
  .controls {
    width: 74px;
    display: flex;
    flex-direction: column;
    align-items: center;
    --wails-draggable: drag;
  }
  .checker {
    align-self: flex-end;
    display: grid;
    grid-template-columns: repeat(3, 4px);
    grid-auto-rows: 4px;
    gap: 1px;
    margin-bottom: 14px;
  }
  .checker span {
    background: rgba(0, 0, 0, 0.25);
  }
  .checker span.on {
    background: var(--accent);
  }

  .rocker {
    width: 56px;
    border-radius: 28px;
    background: linear-gradient(180deg, var(--btn-face-hi), var(--btn-face-lo));
    box-shadow:
      inset 0 1px 0 rgba(255, 255, 255, 0.8),
      0 2px 4px rgba(0, 0, 0, 0.35);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    --wails-draggable: no-drag;
  }
  .rk {
    height: 46px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .rk:hover {
    background: rgba(0, 0, 0, 0.05);
  }
  .rk:active {
    background: rgba(0, 0, 0, 0.12);
  }
  .rk-divider {
    height: 1px;
    background: rgba(0, 0, 0, 0.18);
  }
  .rk svg {
    width: 22px;
    height: 22px;
    fill: none;
    stroke: #3a3833;
    stroke-width: 2.4;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  /* horizontal rocker: in-app screen navigation */
  .hrocker {
    position: relative;
    margin-top: 12px;
    width: 56px;
    height: 26px;
    border-radius: 13px;
    background: linear-gradient(180deg, var(--btn-face-hi), var(--btn-face-lo));
    box-shadow:
      inset 0 1px 0 rgba(255, 255, 255, 0.8),
      0 2px 4px rgba(0, 0, 0, 0.35);
    display: flex;
    overflow: hidden;
    --wails-draggable: no-drag;
  }
  .hrocker.dim {
    opacity: 0.5;
  }
  /* exactly equal halves (avoids subpixel rounding from a 1px flex divider) */
  .hk {
    flex: 0 0 50%;
    width: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .hk:hover:not(:disabled) {
    background: rgba(0, 0, 0, 0.05);
  }
  .hk:active:not(:disabled) {
    background: rgba(0, 0, 0, 0.12);
  }
  .hk:disabled {
    cursor: default;
  }
  /* divider overlaid on the centre seam so both buttons stay identical */
  .hk-divider {
    position: absolute;
    top: 0;
    bottom: 0;
    left: 50%;
    width: 1px;
    transform: translateX(-0.5px);
    background: rgba(0, 0, 0, 0.18);
  }
  .hk svg {
    width: 18px;
    height: 18px;
    fill: none;
    stroke: #3a3833;
    stroke-width: 2.4;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  .check {
    margin-top: 14px;
    width: 56px;
    height: 56px;
    border-radius: 50%;
    background: radial-gradient(120% 120% at 35% 25%, #e8814a, var(--accent) 55%, var(--accent-dim));
    box-shadow:
      inset 0 2px 2px rgba(255, 255, 255, 0.45),
      inset 0 -3px 6px rgba(0, 0, 0, 0.3),
      0 3px 6px rgba(0, 0, 0, 0.4);
    display: flex;
    align-items: center;
    justify-content: center;
    --wails-draggable: no-drag;
  }
  .check:active {
    transform: translateY(1px);
  }
  .check svg {
    width: 28px;
    height: 28px;
    fill: none;
    stroke: #fff;
    stroke-width: 3;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .check svg.lock {
    width: 24px;
    height: 24px;
    stroke-width: 2.2;
  }
  .check svg.refresh {
    width: 24px;
    height: 24px;
    stroke-width: 1.8;
  }

  .master {
    font-size: 8.5px;
    letter-spacing: 0.01em;
    white-space: nowrap;
    color: rgba(0, 0, 0, 0.5);
  }
</style>
