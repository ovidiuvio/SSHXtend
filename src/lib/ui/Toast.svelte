<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import {
    CheckCircleIcon,
    HelpCircleIcon,
    InfoIcon,
    XCircleIcon,
  } from "svelte-feather-icons";

  const dispatch = createEventDispatcher<{ action: void }>();

  /** The kind of toast to display. */
  export let kind: "info" | "success" | "error" = "info";

  /** The message to display inside the toast. */
  export let message: string;

  /** An optional action to provide as a button on the toast. */
  export let action = "";
</script>

<div class="toast-box">
  {#if kind === "info"}
    <InfoIcon class="w-5 h-5 text-theme-accent flex-shrink-0" />
  {:else if kind === "success"}
    <CheckCircleIcon class="w-5 h-5 text-theme-success flex-shrink-0" />
  {:else if kind === "error"}
    <XCircleIcon class="w-5 h-5 text-theme-error flex-shrink-0" />
  {:else}
    <HelpCircleIcon class="w-5 h-5 text-theme-accent flex-shrink-0" />
  {/if}

  <p class="ml-3">
    {message}
  </p>

  {#if action}
    <div class="ml-auto">
      <button
        class="h-5 ml-3 px-2 flex items-center text-xs border rounded-md border-theme-border hover:border-theme-fg hover:text-theme-fg transition-colors"
        on:click={() => dispatch("action")}
      >
        {action}
      </button>
    </div>
  {/if}
</div>

<style lang="postcss">
  .toast-box {
    @apply border border-theme-border bg-theme-bg-secondary/80 backdrop-blur-sm;
    @apply p-4 rounded-md flex items-start pointer-events-auto;
    @apply text-sm;
  }
</style>
