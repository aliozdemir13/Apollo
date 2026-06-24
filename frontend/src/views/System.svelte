<script lang="ts">
  import { system, systemMode, topApps } from "../lib/store";
  $: a = $system;
  $: s = a.data;
  $: apps = $topApps.data ?? [];

  function bat(state: string): string {
    switch (state) {
      case "charging":
        return "CHG";
      case "full":
        return "FULL";
      case "discharging":
        return "BATT";
      default:
        return "—";
    }
  }
</script>

<div class="sys">
  {#if $systemMode === "apps"}
    <div class="apps-head lcd-mono">TOP APPS · CPU</div>
    {#if apps.length === 0}
      <div class="dim lcd-mono small">···</div>
    {:else}
      <div class="apps lcd-mono">
        {#each apps as p, i}
          <div class="app">
            <span class="rank">{i + 1}</span>
            <span class="aname">{p.name}</span>
            <span class="acpu">{p.cpu}%</span>
          </div>
        {/each}
      </div>
    {/if}
  {:else if a.loading && !s}
    <div class="dim lcd-mono">···</div>
  {:else if a.error}
    <div class="err lcd-mono">{a.error}</div>
  {:else if s}
    <div class="rows lcd-mono">
      <div class="row">
        <span class="k">CPU</span>
        <span class="bar"><span class="fill" style="width:{s.cpuPercent}%"></span></span>
        <span class="v">{s.cpuPercent}%</span>
      </div>
      <div class="row">
        <span class="k">MEM</span>
        <span class="bar"><span class="fill" style="width:{s.memPercent}%"></span></span>
        <span class="v">{s.memPercent}%</span>
      </div>
      <div class="row">
        <span class="k">{bat(s.batteryState)}</span>
        {#if s.batteryPct >= 0}
          <span class="bar"><span class="fill" style="width:{s.batteryPct}%"></span></span>
          <span class="v">{s.batteryPct}%</span>
        {:else}
          <span class="bar"></span>
          <span class="v dim">N/A</span>
        {/if}
      </div>
    </div>
    <div class="sub lcd-mono">
      {s.memUsedGB}/{s.memTotalGB} GB · UP {s.uptimeHours}h
    </div>
  {/if}
</div>

<style>
  .sys {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
  }
  .rows {
    width: 90%;
    display: flex;
    flex-direction: column;
    gap: 9px;
  }
  .row {
    display: grid;
    grid-template-columns: 42px 1fr 40px;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    letter-spacing: 0.1em;
  }
  .k {
    color: var(--accent);
    text-align: left;
  }
  .v {
    text-align: right;
  }
  .bar {
    height: 8px;
    background: rgba(255, 255, 255, 0.08);
    border-radius: 2px;
    overflow: hidden;
  }
  .fill {
    display: block;
    height: 100%;
    background: var(--lcd-text);
    transition: width 0.4s ease;
  }
  .sub {
    font-size: 10px;
    letter-spacing: 0.12em;
    color: var(--lcd-dim);
  }
  .apps-head {
    font-size: 11px;
    letter-spacing: 0.2em;
    color: var(--accent);
  }
  .apps {
    width: 92%;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .app {
    display: grid;
    grid-template-columns: 14px 1fr 46px;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    letter-spacing: 0.04em;
  }
  .rank {
    color: var(--lcd-dim);
    text-align: right;
  }
  .aname {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .acpu {
    text-align: right;
    color: var(--accent);
  }
  .err {
    font-size: 11px;
    color: var(--accent);
  }
  .dim {
    color: var(--lcd-dim);
  }
</style>
