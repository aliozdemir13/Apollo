<script lang="ts">
  import { weather } from "../lib/store";
  $: a = $weather;
  $: w = a.data;
</script>

<div class="weather">
  {#if a.loading && !w}
    <div class="loading lcd-mono dim">···</div>
  {:else if a.error}
    <div class="err lcd-mono">{a.error}</div>
  {:else if w}
    <div class="label lcd-mono">{w.label}</div>
    <div class="temp lcd-mono">
      {w.temp}<span class="unit">°{w.unit}</span>
    </div>
    <div class="loc lcd-mono">{w.location || "—"}</div>
    <div class="detail lcd-mono">
      <span>FEEL {w.feelsLike}°</span>
      <span>HUM {w.humidity}%</span>
      <span>WND {w.windSpeed}</span>
    </div>
  {/if}
</div>

<style>
  .weather {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 3px;
  }
  .label {
    font-size: 13px;
    letter-spacing: 0.22em;
    color: var(--accent);
  }
  .temp {
    font-size: 56px;
    font-weight: 300;
    line-height: 1;
  }
  .unit {
    font-size: 22px;
    color: var(--lcd-dim);
  }
  .loc {
    font-size: 12px;
    letter-spacing: 0.16em;
    text-transform: uppercase;
  }
  .detail {
    margin-top: 6px;
    display: flex;
    gap: 10px;
    font-size: 10px;
    letter-spacing: 0.1em;
    color: var(--lcd-dim);
  }
  .err {
    font-size: 11px;
    color: var(--accent);
    text-align: center;
    max-width: 180px;
  }
  .dim {
    color: var(--lcd-dim);
  }
</style>
