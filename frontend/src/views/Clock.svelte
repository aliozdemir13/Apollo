<script lang="ts">
  import { now, weather } from "../lib/store";
  import { clockTime, clockDate } from "../lib/format";

  $: time = clockTime($now);
  $: date = clockDate($now);
  $: w = $weather.data;
</script>

<div class="clock">
  {#if w}
    <div class="wx lcd-mono">
      {w.label} <span class="accent">{w.temp}°</span>
    </div>
  {:else}
    <div class="wx lcd-mono dim">— —</div>
  {/if}

  <div class="time lcd-mono">{time}</div>

  <div class="date lcd-mono">{date}</div>
</div>

<style>
  .clock {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
  }
  .wx {
    font-size: 12px;
    letter-spacing: 0.18em;
    color: var(--lcd-text);
  }
  .wx.dim {
    color: var(--lcd-dim);
  }
  .accent {
    color: var(--accent);
  }
  .time {
    font-size: 58px;
    font-weight: 300;
    line-height: 1;
    letter-spacing: 0.02em;
  }
  .date {
    font-size: 13px;
    letter-spacing: 0.22em;
    color: var(--lcd-text);
  }
</style>
