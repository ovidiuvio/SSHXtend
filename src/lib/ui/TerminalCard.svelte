<!-- @component Individual terminal preview card for the terminal selector -->
<script lang="ts">
  import type { WsWinsize } from "$lib/protocol";
  
  export let id: number;
  export let winsize: WsWinsize;
  export let title: string = "Terminal";
  export let thumbnail: string | null = null;
  export let index: number;
  export let selected: boolean = false;
  export let focused: boolean = false;
  
  // Get keyboard shortcut key for this terminal
  function getShortcutKey(idx: number): string {
    if (idx < 9) return (idx + 1).toString();
    if (idx === 9) return "0";
    const extraKeys = ["Q", "W", "E", "R"];
    if (idx < 14) return extraKeys[idx - 10];
    return "";
  }
  
  $: shortcutKey = getShortcutKey(index);
</script>

<button
  class="terminal-card"
  class:selected
  class:focused
  on:click
  on:mouseenter
  on:mouseleave
  tabindex={selected ? 0 : -1}
>
  <!-- Terminal number badge -->
  {#if shortcutKey}
    <div class="terminal-badge">
      {shortcutKey}
    </div>
  {/if}
  
  <!-- Terminal thumbnail preview -->
  {#if thumbnail}
    <div class="terminal-thumbnail">
      <img src={thumbnail} alt="Terminal preview" />
    </div>
  {:else}
    <div class="terminal-thumbnail-placeholder">
      <div class="placeholder-icon">⌨️</div>
    </div>
  {/if}
  
  <!-- Terminal info -->
  <div class="terminal-info">
    <div class="terminal-title" title={title}>
      {title}
    </div>
    <div class="terminal-meta">
      {winsize.cols}×{winsize.rows}
    </div>
  </div>
  
  <!-- Status indicator -->
  <div class="terminal-status">
    {#if focused}
      <span class="status-dot active" title="Active"></span>
    {:else}
      <span class="status-dot idle" title="Idle"></span>
    {/if}
  </div>
</button>

<style lang="postcss">
  .terminal-card {
    @apply relative bg-theme-bg border-2 border-theme-border rounded-lg p-3;
    @apply cursor-pointer transition-all duration-200 w-full;
    @apply flex flex-col gap-2 min-h-[160px];
    @apply hover:border-blue-500 hover:scale-105;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-transparent;
  }
  
  .terminal-card.selected {
    @apply border-blue-500 scale-105;
    box-shadow: 0 0 20px rgba(59, 130, 246, 0.3);
  }
  
  .terminal-card.focused {
    background: rgb(var(--color-background-secondary));
  }
  
  .terminal-badge {
    @apply absolute -top-2 -right-2 bg-blue-500 text-white;
    @apply rounded-full w-6 h-6 flex items-center justify-center;
    @apply font-bold text-xs;
  }
  
  .terminal-thumbnail {
    @apply w-full h-24 bg-black rounded overflow-hidden mb-2;
    @apply flex items-center justify-center;
    @apply border border-theme-border;
  }
  
  .terminal-thumbnail img {
    @apply w-full h-full object-contain;
    image-rendering: auto;
  }
  
  .terminal-thumbnail-placeholder {
    @apply w-full h-24 rounded mb-2;
    @apply flex items-center justify-center;
    @apply border border-theme-border;
    background: rgb(var(--color-background-secondary));
  }
  
  .placeholder-icon {
    @apply text-2xl opacity-50;
  }
  
  .terminal-info {
    @apply flex-1;
  }
  
  .terminal-title {
    @apply font-mono text-sm text-theme-fg;
    @apply overflow-hidden text-ellipsis whitespace-nowrap;
  }
  
  .terminal-meta {
    @apply font-mono text-xs text-theme-fg-secondary mt-1;
  }
  
  .terminal-status {
    @apply flex items-center justify-end;
  }
  
  .status-dot {
    @apply w-2 h-2 rounded-full;
  }
  
  .status-dot.active {
    @apply bg-green-500;
    animation: pulse 2s infinite;
  }
  
  .status-dot.idle {
    @apply bg-gray-500;
  }
  
  @keyframes pulse {
    0%, 100% {
      opacity: 1;
    }
    50% {
      opacity: 0.5;
    }
  }
</style>