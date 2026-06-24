<script lang="ts">
  import { onMount } from "svelte";
  import { settings, saveSettings, settingsOpen, refreshMFA } from "./lib/store";
  import {
    MFAListAccounts,
    MFAAddAccount,
    MFARemoveAccount,
    MFASetPin,
    MFAClearPin,
    MFAGetStatus,
  } from "../wailsjs/go/main/App";

  const VIEW_LABELS: Record<string, string> = {
    clock: "Clock & date",
    weather: "Weather",
    system: "System stats",
    github: "GitHub PRs",
    teams: "Teams unread",
    totp: "2FA (Salesforce)",
  };
  const ALL_VIEWS = ["clock", "weather", "system", "github", "teams", "totp"];

  // ---- MFA management (operates immediately, separate from Save) ----
  let mfaAccounts: any[] = [];
  let mfaHasPin = false;
  let pinInput = "";
  let newLabel = "";
  let newSecret = "";
  let mfaMsg = "";

  async function loadMFA() {
    mfaAccounts = await MFAListAccounts();
    const st = await MFAGetStatus();
    mfaHasPin = st.hasPin;
  }

  async function setPin() {
    mfaMsg = "";
    if (!/^\d{4,}$/.test(pinInput)) {
      mfaMsg = "PIN must be at least 4 digits";
      return;
    }
    try {
      await MFASetPin(pinInput);
      pinInput = "";
      mfaMsg = "PIN saved";
      await loadMFA();
      await refreshMFA();
    } catch (e: any) {
      mfaMsg = String(e?.message || e);
    }
  }

  async function clearPin() {
    await MFAClearPin();
    mfaMsg = "PIN removed";
    await loadMFA();
    await refreshMFA();
  }

  async function addOrg() {
    mfaMsg = "";
    if (!newLabel.trim() || !newSecret.trim()) {
      mfaMsg = "Name and key are required";
      return;
    }
    try {
      await MFAAddAccount(newLabel.trim(), "Salesforce", newSecret.trim());
      newLabel = "";
      newSecret = "";
      mfaMsg = "Org added";
      await loadMFA();
      await refreshMFA();
    } catch (e: any) {
      mfaMsg = String(e?.message || e);
    }
  }

  async function removeOrg(id: string) {
    await MFARemoveAccount(id);
    await loadMFA();
    await refreshMFA();
  }

  onMount(loadMFA);

  // Local editable copy.
  let s = JSON.parse(JSON.stringify($settings || {}));
  s.units = s.units || "celsius";
  s.teamsSource = s.teamsSource || "graph";
  s.views = s.views?.length ? s.views : [...ALL_VIEWS];

  let reposText = (s.githubRepos || []).join("\n");
  let favText = (s.teamsFavorites || []).join("\n");
  let saving = false;

  function toggleView(v: string) {
    if (s.views.includes(v)) {
      s.views = s.views.filter((x: string) => x !== v);
    } else {
      // keep canonical order
      s.views = ALL_VIEWS.filter((x) => x === v || s.views.includes(x));
    }
  }

  async function onSave() {
    saving = true;
    s.githubRepos = reposText
      .split("\n")
      .map((x: string) => x.trim())
      .filter(Boolean);
    s.teamsFavorites = favText
      .split("\n")
      .map((x: string) => x.trim())
      .filter(Boolean);
    if (!s.views.length) s.views = [...ALL_VIEWS];
    try {
      await saveSettings(s);
      settingsOpen.set(false);
    } finally {
      saving = false;
    }
  }
</script>

<div class="sheet">
  <div class="bar">
    <span class="title lcd-mono">SETTINGS</span>
    <button class="x" on:click={() => settingsOpen.set(false)}>✕</button>
  </div>

  <div class="scroll">
    <section>
      <h3>Weather</h3>
      <label>Location
        <input bind:value={s.locationName} placeholder="e.g. Munich (blank = auto)" />
      </label>
      <label>Units
        <select bind:value={s.units}>
          <option value="celsius">Celsius °C</option>
          <option value="fahrenheit">Fahrenheit °F</option>
        </select>
      </label>
    </section>

    <section>
      <h3>Views</h3>
      <div class="views">
        {#each ALL_VIEWS as v}
          <label class="chk">
            <input type="checkbox" checked={s.views.includes(v)} on:change={() => toggleView(v)} />
            {VIEW_LABELS[v]}
          </label>
        {/each}
      </div>
    </section>

    <section>
      <h3>GitHub</h3>
      <label>Token (repo scope)
        <input type="password" bind:value={s.githubToken} placeholder="ghp_…" />
      </label>
      <label>Repos (one per line, owner/name)
        <textarea rows="3" bind:value={reposText} placeholder="anthropics/anthropic-sdk-python"></textarea>
      </label>
      <label>Only my PRs (optional GitHub login)
        <input bind:value={s.githubLogin} placeholder="octocat" />
      </label>
    </section>

    <section>
      <h3>Teams</h3>
      <label>Source
        <select bind:value={s.teamsSource}>
          <option value="local">Local — macOS notifications (no keys)</option>
          <option value="graph">Microsoft Graph (Azure app)</option>
        </select>
      </label>

      {#if s.teamsSource === "local"}
        <div class="note">
          Reads delivered Teams notifications from macOS. Requires <b>Full Disk
          Access</b> for this app (System Settings → Privacy &amp; Security → Full
          Disk Access). macOS only.
        </div>
      {:else}
        <label>Azure App (client) ID
          <input bind:value={s.teamsClientId} placeholder="00000000-0000-…" />
        </label>
        <label>Tenant ID
          <input bind:value={s.teamsTenantId} placeholder="common / organizations / GUID" />
        </label>
      {/if}

      <label>Favorites filter (one per line, optional)
        <textarea rows="2" bind:value={favText} placeholder="Team name or person"></textarea>
      </label>
    </section>

    <section>
      <h3>2FA — Salesforce orgs</h3>

      <label>Unlock PIN ({mfaHasPin ? "set" : "not set"})
        <div class="inline">
          <input
            type="password"
            inputmode="numeric"
            bind:value={pinInput}
            placeholder="4-digit PIN"
          />
          <button class="mini" on:click={setPin}>{mfaHasPin ? "Change" : "Set"}</button>
          {#if mfaHasPin}
            <button class="mini ghost" on:click={clearPin}>Remove</button>
          {/if}
        </div>
      </label>
      <div class="note">2FA screen auto-locks after 5 min and needs this PIN.</div>

      {#if mfaAccounts.length}
        <div class="orgs">
          {#each mfaAccounts as a}
            <div class="org-row">
              <span class="org-name">{a.label}</span>
              <span class="org-issuer">{a.issuer}</span>
              <button class="mini ghost" on:click={() => removeOrg(a.id)}>✕</button>
            </div>
          {/each}
        </div>
      {/if}

      <label>Org name
        <input bind:value={newLabel} placeholder="e.g. ACME Prod" />
      </label>
      <label>Authenticator key (from Salesforce "Can't scan?")
        <input bind:value={newSecret} placeholder="base32 secret, spaces ok" />
      </label>
      <button class="mini add" on:click={addOrg}>+ Add org</button>

      {#if mfaMsg}<div class="note accent">{mfaMsg}</div>{/if}
    </section>

    {#if s.configPath}
      <div class="path">config: {s.configPath}</div>
    {/if}
  </div>

  <div class="footer">
    <button class="cancel" on:click={() => settingsOpen.set(false)}>Cancel</button>
    <button class="save" on:click={onSave} disabled={saving}>
      {saving ? "Saving…" : "Save"}
    </button>
  </div>
</div>

<style>
  .sheet {
    position: absolute;
    inset: 0;
    background: #14130f;
    border-radius: 14px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    z-index: 20;
  }
  .bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 14px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.07);
  }
  .title {
    font-size: 12px;
    letter-spacing: 0.22em;
    color: var(--accent);
  }
  .x {
    color: var(--lcd-dim);
    font-size: 14px;
  }
  .scroll {
    flex: 1;
    overflow-y: auto;
    padding: 10px 14px;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  section {
    display: flex;
    flex-direction: column;
    gap: 7px;
  }
  h3 {
    margin: 0;
    font-size: 10px;
    letter-spacing: 0.18em;
    text-transform: uppercase;
    color: var(--lcd-dim);
  }
  label {
    display: flex;
    flex-direction: column;
    gap: 3px;
    font-size: 11px;
    color: var(--lcd-text);
  }
  input,
  textarea,
  select {
    background: #0a0a0a;
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: 6px;
    color: var(--lcd-text);
    padding: 6px 8px;
    font-size: 12px;
    font-family: inherit;
    outline: none;
  }
  input:focus,
  textarea:focus,
  select:focus {
    border-color: var(--accent);
  }
  textarea {
    resize: vertical;
  }
  .views {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }
  .chk {
    flex-direction: row;
    align-items: center;
    gap: 8px;
    font-size: 12px;
  }
  .chk input {
    width: auto;
    accent-color: var(--accent);
  }
  .path {
    font-size: 9px;
    color: var(--lcd-dim);
    word-break: break-all;
  }
  .inline {
    display: flex;
    gap: 6px;
    align-items: center;
  }
  .inline input {
    flex: 1;
  }
  .mini {
    padding: 6px 10px;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.1);
    color: var(--lcd-text);
    font-size: 11px;
    white-space: nowrap;
  }
  .mini.ghost {
    background: rgba(255, 255, 255, 0.05);
    color: var(--lcd-dim);
  }
  .mini.add {
    align-self: flex-start;
    background: rgba(224, 113, 47, 0.85);
    color: #1a1106;
  }
  .note {
    font-size: 9px;
    letter-spacing: 0.06em;
    color: var(--lcd-dim);
  }
  .note.accent {
    color: var(--accent);
  }
  .orgs {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .org-row {
    display: grid;
    grid-template-columns: 1fr auto 22px;
    align-items: center;
    gap: 8px;
    padding: 5px 6px;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.05);
    font-size: 12px;
  }
  .org-name {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .org-issuer {
    font-size: 9px;
    color: var(--lcd-dim);
    letter-spacing: 0.1em;
  }
  .footer {
    display: flex;
    gap: 10px;
    padding: 10px 14px;
    border-top: 1px solid rgba(255, 255, 255, 0.07);
  }
  .footer button {
    flex: 1;
    padding: 8px;
    border-radius: 8px;
    font-size: 12px;
    letter-spacing: 0.06em;
  }
  .cancel {
    background: rgba(255, 255, 255, 0.08);
    color: var(--lcd-text);
  }
  .save {
    background: var(--accent);
    color: #1a1106;
    font-weight: 600;
  }
  .save:disabled {
    opacity: 0.6;
  }
</style>
