<script lang="ts">
  import {
    github,
    githubMode,
    githubReviews,
    githubWorkflows,
    openExternal,
    settingsOpen,
  } from "../lib/store";
  import { ago } from "../lib/format";

  $: prs = $github.data?.prs ?? [];
  $: prErrors = $github.data?.errors ?? [];
  $: reviews = $githubReviews.data?.prs ?? [];
  $: runs = $githubWorkflows.data?.runs ?? [];

  const MODE_LABEL: Record<string, string> = {
    prs: "OPEN PRS",
    reviews: "TO REVIEW",
    workflows: "WORKFLOWS",
    errors: "REPO ERRORS",
  };

  function shortRepo(full: string): string {
    const parts = (full || "").split("/");
    return parts[parts.length - 1] || full;
  }

  // GitHub Actions run → short badge + colour class.
  function runBadge(r: any): { text: string; cls: string } {
    if (r.status !== "completed") return { text: r.status.replace("_", " "), cls: "run" };
    switch (r.conclusion) {
      case "success":
        return { text: "OK", cls: "ok" };
      case "failure":
        return { text: "FAIL", cls: "fail" };
      case "cancelled":
        return { text: "CANCEL", cls: "dimbadge" };
      default:
        return { text: (r.conclusion || "—").toUpperCase(), cls: "dimbadge" };
    }
  }
</script>

<div class="gh">
  <div class="mode lcd-mono">{MODE_LABEL[$githubMode]}</div>

  {#if $githubMode === "prs"}
    {#if $github.loading && !$github.data}
      <div class="dim lcd-mono">···</div>
    {:else if $github.error}
      <div class="empty lcd-mono">
        <div class="err">{$github.error}</div>
        <button class="link" on:click={() => settingsOpen.set(true)}>SET TOKEN ›</button>
      </div>
    {:else if prs.length === 0}
      <div class="dim lcd-mono small">NO OPEN PRS</div>
    {:else}
      <div class="list">
        {#each prs.slice(0, 6) as pr}
          <button class="item lcd-mono" on:click={() => openExternal(pr.url)}>
            <span class="repo">{shortRepo(pr.repo)}</span>
            <span class="title">{pr.title}</span>
            <span class="meta">
              {#if pr.draft}<span class="draft">DRAFT</span>{/if}
              <span class="num">#{pr.number}</span>
              <span class="age">{ago(pr.updatedAt)}</span>
            </span>
          </button>
        {/each}
      </div>
    {/if}

  {:else if $githubMode === "reviews"}
    {#if $githubReviews.loading && !$githubReviews.data}
      <div class="dim lcd-mono">···</div>
    {:else if $githubReviews.error}
      <div class="err lcd-mono">{$githubReviews.error}</div>
    {:else if reviews.length === 0}
      <div class="dim lcd-mono small">NOTHING TO REVIEW</div>
    {:else}
      <div class="list">
        {#each reviews.slice(0, 6) as pr}
          <button class="item lcd-mono" on:click={() => openExternal(pr.url)}>
            <span class="repo">{shortRepo(pr.repo)}</span>
            <span class="title">{pr.title}</span>
            <span class="meta">
              <span class="by">@{pr.author}</span>
              <span class="num">#{pr.number}</span>
              <span class="age">{ago(pr.updatedAt)}</span>
            </span>
          </button>
        {/each}
      </div>
    {/if}

  {:else if $githubMode === "workflows"}
    {#if $githubWorkflows.loading && !$githubWorkflows.data}
      <div class="dim lcd-mono">···</div>
    {:else if $githubWorkflows.error}
      <div class="err lcd-mono">{$githubWorkflows.error}</div>
    {:else if runs.length === 0}
      <div class="dim lcd-mono small">NO RUNS</div>
    {:else}
      <div class="list">
        {#each runs.slice(0, 6) as r}
          <button class="item lcd-mono" on:click={() => openExternal(r.url)}>
            <span class="wfhead">
              <span class="repo">{shortRepo(r.repo)}</span>
              <span class="badge {runBadge(r).cls}">{runBadge(r).text}</span>
            </span>
            <span class="title">{r.name}</span>
            <span class="meta">
              <span class="branch">{r.branch}</span>
              <span class="age">{ago(r.updatedAt)}</span>
            </span>
          </button>
        {/each}
      </div>
    {/if}

  {:else}
    <!-- errors -->
    {#if prErrors.length === 0}
      <div class="dim lcd-mono small">NO REPO ERRORS</div>
    {:else}
      <div class="list">
        {#each prErrors as e}
          <div class="errrow lcd-mono">{e}</div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<style>
  .gh {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 7px;
  }
  .mode {
    font-size: 11px;
    letter-spacing: 0.22em;
    color: var(--accent);
  }
  .list {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 5px;
    max-height: 135px;
    overflow-y: auto;
  }
  .item {
    text-align: left;
    display: flex;
    flex-direction: column;
    gap: 1px;
    padding: 4px 6px;
    border-radius: 5px;
    background: rgba(255, 255, 255, 0.04);
  }
  .item:hover {
    background: rgba(224, 113, 47, 0.14);
  }
  .repo {
    font-size: 9px;
    letter-spacing: 0.12em;
    color: var(--accent);
  }
  .title {
    font-size: 11px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 210px;
  }
  .meta {
    display: flex;
    gap: 6px;
    font-size: 9px;
    color: var(--lcd-dim);
    letter-spacing: 0.08em;
  }
  .draft {
    color: var(--lcd-dim);
  }
  .wfhead {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }
  .badge {
    font-size: 9px;
    letter-spacing: 0.08em;
    padding: 1px 5px;
    border-radius: 3px;
  }
  .badge.ok {
    color: #0c1a0c;
    background: #6fbf6a;
  }
  .badge.fail {
    color: #1a0c06;
    background: var(--accent);
  }
  .badge.run {
    color: #1a1606;
    background: #d8c24a;
  }
  .badge.dimbadge {
    color: var(--lcd-dim);
    background: rgba(255, 255, 255, 0.08);
  }
  .errrow {
    font-size: 10px;
    color: var(--accent);
    background: rgba(224, 113, 47, 0.1);
    padding: 5px 6px;
    border-radius: 5px;
    word-break: break-word;
  }
  .small {
    font-size: 11px;
    letter-spacing: 0.16em;
  }
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
  }
  .err {
    font-size: 11px;
    color: var(--accent);
    text-align: center;
    max-width: 200px;
  }
  .link {
    font-size: 10px;
    letter-spacing: 0.14em;
    color: var(--lcd-text);
  }
  .dim {
    color: var(--lcd-dim);
  }
</style>
