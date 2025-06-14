<script lang="ts">
  import { fade } from "svelte/transition";

  export let status: "connected" | "no-server" | "no-shell";

  export let serverLatency: number | null;
  export let shellLatency: number | null;

  function displayLatency(latency: number) {
    if (latency < 1) {
      return "1 ms";
    } else if (latency <= 950) {
      return `${Math.round(latency)} ms`;
    } else {
      return `${(latency / 1000).toFixed(1)} s`;
    }
  }

  function colorLatency(latency: number | null) {
    if (latency === null) {
      return "";
    } else if (latency < 80) {
      return "text-theme-success";
    } else if (latency < 300) {
      return "text-theme-warning";
    } else {
      return "text-theme-error";
    }
  }
</script>

<div
  class="relative panel p-4"
  in:fade|local={{ duration: 100 }}
  out:fade|local={{ duration: 75 }}
>
  <div class="absolute left-[calc(50%-8px)] top-[-16px] w-4 h-4">
    <svg viewBox="0 0 16 16">
      <path
        d="M 0 12 L 8 0 L 16 12 Z"
        fill="rgb(var(--color-background-secondary))"
        stroke="rgb(var(--color-border))"
      />
    </svg>
  </div>

  <h2 class="font-medium mb-1 text-center">Network</h2>
  <p class="text-theme-fg-muted text-sm text-center">
    {#if status === "connected"}
      {#if serverLatency === null || shellLatency === null}
        Connected, estimating latency…
      {:else}
        Total latency: {displayLatency(serverLatency + shellLatency)}
      {/if}
    {:else}
      You are currently disconnected.
    {/if}
  </p>

  <div class="flex justify-between items-center mt-6">
    <div class="ball filled" />
    <div class="border-t-2 border-dashed border-theme-border-secondary w-32" />
    <div class="ball" class:filled={status !== "no-server"} />
    <div class="border-t-2 border-dashed border-theme-border-secondary w-32" />
    <div class="ball" class:filled={status === "connected"} />
  </div>

  <div class="flex justify-between items-center mt-2.5">
    <p class="text-xs text-theme-fg-secondary w-8">You</p>

    {#if status === "connected"}
      <p class="text-xs w-14 text-left {colorLatency(serverLatency)}">
        {#if serverLatency !== null}
          ~{displayLatency(serverLatency)}
        {/if}
      </p>
    {/if}

    <p class="text-xs text-theme-fg-secondary">Server</p>

    {#if status === "connected"}
      <p class="text-xs w-14 text-right {colorLatency(shellLatency)}">
        {#if shellLatency !== null}
          ~{displayLatency(shellLatency)}
        {/if}
      </p>
    {/if}

    <p class="text-xs text-theme-fg-secondary w-8 text-right">Shell</p>
  </div>
</div>

<style lang="postcss">
  .ball {
    @apply rounded-full w-4 h-4;
  }

  .ball.filled {
    @apply border border-theme-fg-secondary bg-theme-bg-tertiary;
  }

  .ball:not(.filled) {
    @apply border-2 border-theme-border-secondary;
  }
</style>
