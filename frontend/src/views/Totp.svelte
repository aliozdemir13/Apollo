<script lang="ts">
  import { onMount } from "svelte";
  import {
    now,
    mfaHasPin,
    mfaUnlocked,
    mfaCodes,
    mfaPin,
    mfaError,
    mfaAccountCount,
    pinPush,
    pinBackspace,
    copyCode,
    refreshMFA,
    settingsOpen,
  } from "../lib/store";

  // Shared 30s TOTP window countdown, computed locally from the clock tick.
  $: secs = 30 - (Math.floor($now.getTime() / 1000) % 30);

  let copiedId = "";
  let copyTimer: number | undefined;
  function doCopy(id: string, code: string) {
    copyCode(code);
    copiedId = id;
    clearTimeout(copyTimer);
    copyTimer = window.setTimeout(() => (copiedId = ""), 1200);
  }

  function fmt(code: string): string {
    return code && code.length === 6 ? code.slice(0, 3) + " " + code.slice(3) : code;
  }

  function onKey(e: KeyboardEvent) {
    if ($mfaUnlocked) return;
    if (/^[0-9]$/.test(e.key)) {
      pinPush(e.key);
      e.preventDefault();
    } else if (e.key === "Backspace") {
      pinBackspace();
      e.preventDefault();
    }
  }

  onMount(refreshMFA);
</script>

<svelte:window on:keydown={onKey} />

<div class="totp">
  {#if !$mfaHasPin}
    <!-- No PIN configured yet -->
    <div class="empty lcd-mono">
      <div class="msg">2FA LOCKED</div>
      <div class="hint">set a PIN in settings</div>
      <button class="link" on:click={() => settingsOpen.set(true)}>SETTINGS ›</button>
    </div>
  {:else if !$mfaUnlocked}
    <!-- Lock screen: PIN entry -->
    <div class="lock lcd-mono">
      <div class="dots">
        {#each [0, 1, 2, 3] as i}
          <span class="dot" class:filled={$mfaPin.length > i}></span>
        {/each}
      </div>
      {#if $mfaError}<div class="pinerr">{$mfaError}</div>{/if}
      <div class="keypad">
        {#each ["1", "2", "3", "4", "5", "6", "7", "8", "9"] as d}
          <button class="key" on:click={() => pinPush(d)}>{d}</button>
        {/each}
        <button class="key blank" disabled></button>
        <button class="key" on:click={() => pinPush("0")}>0</button>
        <button class="key" on:click={pinBackspace}>⌫</button>
      </div>
    </div>
  {:else if $mfaAccountCount === 0}
    <div class="empty lcd-mono">
      <div class="msg">NO ORGS</div>
      <button class="link" on:click={() => settingsOpen.set(true)}>ADD ONE ›</button>
    </div>
  {:else}
    <!-- Unlocked: codes. Each tile's background bar drains over the 30s window. -->
    <div class="codes lcd-mono">
      <div class="list">
        {#each $mfaCodes as c}
          <button class="tile" on:click={() => doCopy(c.id, c.code)} title="Click to copy">
            <span class="bar" class:warn={secs <= 5} style="width:{(secs / 30) * 100}%"></span>
            <span class="content">
              <span class="top">
                <span class="org">{c.label}</span>
                <span class="secs" class:warn={secs <= 5}>{secs}s</span>
              </span>
              {#if c.error}
                <span class="codeerr">ERROR</span>
              {:else}
                <span class="code">{copiedId === c.id ? "COPIED" : fmt(c.code)}</span>
              {/if}
            </span>
          </button>
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .totp {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  /* lock screen */
  .lock {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
  }
  .dots {
    display: flex;
    gap: 12px;
  }
  .dots .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    border: 1px solid var(--lcd-text);
  }
  .dots .dot.filled {
    background: var(--accent);
    border-color: var(--accent);
  }
  .pinerr {
    font-size: 10px;
    letter-spacing: 0.18em;
    color: var(--accent);
  }
  .keypad {
    display: grid;
    grid-template-columns: repeat(3, 40px);
    gap: 6px;
  }
  .key {
    height: 30px;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.06);
    color: var(--lcd-text);
    font-size: 15px;
  }
  .key:hover:not(.blank) {
    background: rgba(224, 113, 47, 0.18);
  }
  .key:active:not(.blank) {
    background: rgba(224, 113, 47, 0.3);
  }
  .key.blank {
    background: none;
    cursor: default;
  }

  /* codes */
  .codes {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
  }
  .list {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 150px;
    overflow-y: auto;
  }
  .tile {
    position: relative;
    width: 100%;
    border-radius: 9px;
    background: rgba(255, 255, 255, 0.05);
    overflow: hidden;
    text-align: left;
    padding: 0;
  }
  .tile:hover {
    background: rgba(255, 255, 255, 0.09);
  }
  /* draining countdown bar that fills the whole tile as a background */
  .bar {
    position: absolute;
    top: 0;
    left: 0;
    bottom: 0;
    background: rgba(224, 113, 47, 0.28);
    transition: width 1s linear;
    z-index: 0;
  }
  .bar.warn {
    background: rgba(224, 113, 47, 0.5);
  }
  .content {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    gap: 3px;
    padding: 9px 12px;
  }
  .top {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 8px;
  }
  .org {
    font-size: 11px;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--lcd-text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .secs {
    font-size: 10px;
    letter-spacing: 0.08em;
    color: var(--lcd-dim);
    flex-shrink: 0;
  }
  .secs.warn {
    color: var(--accent);
  }
  .code {
    font-size: 26px;
    letter-spacing: 0.16em;
    color: var(--lcd-text);
    line-height: 1.05;
  }
  .codeerr {
    font-size: 13px;
    color: var(--accent);
  }

  /* shared */
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
  }
  .msg {
    font-size: 12px;
    letter-spacing: 0.2em;
    color: var(--accent);
  }
  .hint {
    font-size: 10px;
    letter-spacing: 0.12em;
    color: var(--lcd-dim);
  }
  .link {
    font-size: 10px;
    letter-spacing: 0.14em;
    color: var(--lcd-text);
  }
</style>
