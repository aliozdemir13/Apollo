<script lang="ts">
  import {
    teams,
    teamsLoggedIn,
    teamsLoggingIn,
    deviceCode,
    settings,
    settingsOpen,
  } from "../lib/store";
  import { ago } from "../lib/format";

  $: a = $teams;
  $: r = a.data;
  $: chats = r?.unreadChats ?? [];
  $: isLocal = $settings?.teamsSource === "local";
  // Local mode (macOS notifications) needs no sign-in; Graph mode needs a client id.
  $: configured = isLocal || !!$settings?.teamsClientId;
</script>

<div class="teams">
  {#if !configured}
    <div class="empty lcd-mono">
      <div class="msg">TEAMS NOT SET UP</div>
      <button class="link" on:click={() => settingsOpen.set(true)}>CONFIGURE ›</button>
    </div>
  {:else if !isLocal && $deviceCode}
    <div class="auth lcd-mono">
      <div class="msg">SIGN IN AT</div>
      <div class="url">{$deviceCode.verificationUrl}</div>
      <div class="msg">CODE</div>
      <div class="code">{$deviceCode.userCode}</div>
      <div class="hint">waiting…</div>
    </div>
  {:else if !isLocal && !$teamsLoggedIn}
    <div class="empty lcd-mono">
      <div class="msg">SIGNED OUT</div>
      <div class="hint">press orange button to sign in</div>
      {#if $teamsLoggingIn}<div class="hint">starting…</div>{/if}
    </div>
  {:else if a.loading && !r}
    <div class="dim lcd-mono">···</div>
  {:else if a.error}
    <div class="err lcd-mono">{a.error}</div>
  {:else}
    <div class="head lcd-mono">
      <span class="count">{chats.length}</span>
      <span class="lbl">UNREAD</span>
    </div>
    {#if chats.length === 0}
      <div class="dim lcd-mono small">ALL CAUGHT UP</div>
    {:else}
      <div class="list">
        {#each chats.slice(0, 6) as c}
          <div class="item lcd-mono">
            <span class="name">{c.name}</span>
            <span class="preview">{c.from ? c.from + ": " : ""}{c.preview}</span>
            <span class="age">{ago(c.timestamp)}</span>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<style>
  .teams {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
  }
  .head {
    display: flex;
    align-items: baseline;
    gap: 8px;
  }
  .count {
    font-size: 34px;
    font-weight: 300;
    color: var(--accent);
    line-height: 1;
  }
  .lbl {
    font-size: 12px;
    letter-spacing: 0.2em;
  }
  .list {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 5px;
    max-height: 120px;
    overflow-y: auto;
  }
  .item {
    text-align: left;
    display: flex;
    flex-direction: column;
    gap: 1px;
    padding: 3px 4px;
    border-radius: 4px;
    background: rgba(255, 255, 255, 0.03);
  }
  .name {
    font-size: 11px;
    color: var(--accent);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 200px;
  }
  .preview {
    font-size: 10px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 200px;
    color: var(--lcd-text);
  }
  .age {
    font-size: 9px;
    color: var(--lcd-dim);
  }
  .auth {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
  }
  .url {
    font-size: 12px;
    color: var(--lcd-text);
    letter-spacing: 0.04em;
  }
  .code {
    font-size: 24px;
    color: var(--accent);
    letter-spacing: 0.18em;
  }
  .msg {
    font-size: 11px;
    letter-spacing: 0.16em;
    color: var(--lcd-dim);
  }
  .hint {
    font-size: 10px;
    letter-spacing: 0.12em;
    color: var(--lcd-dim);
  }
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
  }
  .link {
    font-size: 10px;
    letter-spacing: 0.14em;
    color: var(--lcd-text);
  }
  .small {
    font-size: 11px;
    letter-spacing: 0.16em;
  }
  .err {
    font-size: 11px;
    color: var(--accent);
    text-align: center;
    max-width: 190px;
  }
  .dim {
    color: var(--lcd-dim);
  }
</style>
