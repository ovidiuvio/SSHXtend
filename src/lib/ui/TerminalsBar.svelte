<!-- @component Positionable terminals bar for quick terminal switching -->
<script lang="ts">
  import { createEventDispatcher, onMount } from "svelte";
  import { fade, slide } from "svelte/transition";
  import { LayersIcon, LockIcon, UnlockIcon } from "svelte-feather-icons";
  import type { WsWinsize } from "$lib/protocol";
  
  export let shells: [number, WsWinsize][] = [];
  export let focusedTerminals: number[] = [];
  export let terminalTitles: Record<number, string> = {};
  export let terminalThumbnails: Record<number, string | null> = {};
  export let position: "top" | "bottom" | "left" | "right" = "bottom";
  export let pinned: boolean = false;
  export let visible: boolean = true;
  export let mainToolbarPosition: "top" | "bottom" | "left" | "right" = "top";
  
  const dispatch = createEventDispatcher<{
    select: { id: number, winsize: WsWinsize };
    togglePin: void;
    mouseenter: void;
    mouseleave: void;
  }>();
  
  let hoveredTerminal: number | null = null;
  let containerEl: HTMLElement;
  
  // Check for collision with main toolbar
  $: hasCollision = position === mainToolbarPosition;
  
  // Calculate offset when there's a collision
  $: collisionOffset = hasCollision ? (
    position === "top" || position === "bottom" ? "6rem" : "4rem"
  ) : "1rem";

  // Calculate terminal count and thumbnail scaling
  $: terminalCount = shells.length;
  
  // Dynamic thumbnail sizing based on terminal count and screen dimension
  $: isHorizontal = position === "top" || position === "bottom";
  $: maxDimension = isHorizontal ? "90vw" : "90vh";
  

  // Handle window resize to recalculate thumbnail sizes
  let windowWidth = 1920;
  let windowHeight = 1080;

  onMount(() => {
    if (typeof window !== 'undefined') {
      windowWidth = window.innerWidth;
      windowHeight = window.innerHeight;
      
      const handleResize = () => {
        windowWidth = window.innerWidth;
        windowHeight = window.innerHeight;
      };
      
      window.addEventListener('resize', handleResize);
      return () => window.removeEventListener('resize', handleResize);
    }
  });
  
  // Recalculate thumbnail size when window size changes
  $: thumbnailSize = calculateThumbnailSize(terminalCount, isHorizontal, windowWidth, windowHeight);

  function calculateThumbnailSize(count: number, horizontal: boolean, winWidth = 1920, winHeight = 1080) {
    if (count === 0) return { width: 160, height: 96 };
    
    // Base size for single terminal
    const baseWidth = 160;
    const baseHeight = 96;
    const aspectRatio = baseWidth / baseHeight;
    
    // Available space (90% of screen minus padding/header)
    const availableWidth = horizontal ? winWidth * 0.9 - 80 : baseWidth;
    const availableHeight = horizontal ? baseHeight : winHeight * 0.9 - 120;
    
    if (horizontal) {
      // For horizontal layout, fit all terminals in width
      const maxWidthPerTerminal = availableWidth / count;
      const scaledWidth = Math.min(baseWidth, maxWidthPerTerminal - 16); // 16px gap
      const scaledHeight = scaledWidth / aspectRatio;
      
      return {
        width: Math.max(80, scaledWidth), // Minimum 80px width
        height: Math.max(48, scaledHeight) // Minimum 48px height
      };
    } else {
      // For vertical layout, fit all terminals in height
      const maxHeightPerTerminal = availableHeight / count;
      const scaledHeight = Math.min(baseHeight, maxHeightPerTerminal - 16); // 16px gap
      const scaledWidth = scaledHeight * aspectRatio;
      
      return {
        width: Math.max(80, scaledWidth), // Minimum 80px width
        height: Math.max(48, scaledHeight) // Minimum 48px height
      };
    }
  }

  function handleTerminalClick(index: number) {
    if (index >= 0 && index < shells.length) {
      const [id, winsize] = shells[index];
      dispatch('select', { id, winsize });
    }
  }
  
  function handleKeydown(event: KeyboardEvent) {
    // Handle number keys for quick switching
    if (event.key >= '1' && event.key <= '9') {
      const index = parseInt(event.key) - 1;
      if (index < shells.length) {
        handleTerminalClick(index);
        event.preventDefault();
      }
    } else if (event.key === '0' && shells.length > 9) {
      handleTerminalClick(9);
      event.preventDefault();
    }
    
    // Extended keys for terminals 11-14
    const extendedKeys: Record<string, number> = {
      'q': 10, 'Q': 10,
      'w': 11, 'W': 11,
      'e': 12, 'E': 12,
      'r': 13, 'R': 13,
    };
    
    if (event.key in extendedKeys) {
      const index = extendedKeys[event.key];
      if (index < shells.length) {
        handleTerminalClick(index);
        event.preventDefault();
      }
    }
  }
  
</script>

<svelte:window on:keydown={handleKeydown} />

{#if visible}
  <div 
    bind:this={containerEl}
    class="terminals-bar"
    style="--terminals-bar-offset: {collisionOffset}; --max-dimension: {maxDimension}; --thumbnail-width: {thumbnailSize.width}px; --thumbnail-height: {thumbnailSize.height}px;"
    class:horizontal={position === "top" || position === "bottom"}
    class:vertical={position === "left" || position === "right"}
    class:pinned
    class:position-top={position === "top"}
    class:position-bottom={position === "bottom"}
    class:position-left={position === "left"}
    class:position-right={position === "right"}
    on:mouseenter={() => dispatch('mouseenter')}
    on:mouseleave={() => dispatch('mouseleave')}
    transition:slide={{ axis: position === "left" || position === "right" ? "x" : "y", duration: 200 }}
  >
    <!-- Header with controls -->
    <div class="terminals-bar-header">
      <div class="header-info">
        <LayersIcon size="14" strokeWidth={1.5} />
        <span class="terminal-count">{shells.length}</span>
      </div>
      
      <div class="header-controls">
        <button 
          class="control-button"
          on:click={() => dispatch('togglePin')}
          title={pinned ? "Unpin terminals bar" : "Pin terminals bar"}
        >
          {#if pinned}
            <LockIcon size="12" strokeWidth={1.5} />
          {:else}
            <UnlockIcon size="12" strokeWidth={1.5} />
          {/if}
        </button>
      </div>
    </div>
    
    <!-- Terminals list -->
    <div class="terminals-list">
      {#each shells as [id, winsize], index (id)}
        <div
          class="terminal-item"
          class:focused={focusedTerminals.includes(id)}
          class:hovered={hoveredTerminal === id}
          on:click={() => handleTerminalClick(index)}
          on:mouseenter={() => hoveredTerminal = id}
          on:mouseleave={() => hoveredTerminal = null}
          on:keydown={(e) => e.key === 'Enter' && handleTerminalClick(index)}
          role="button"
          tabindex="0"
          title={terminalTitles[id] || `Terminal ${id}`}
        >
          
          <!-- Terminal thumbnail -->
          <div class="terminal-thumbnail">
            {#if terminalThumbnails[id]}
              <img 
                src={terminalThumbnails[id]} 
                alt="Terminal {id}"
                class="thumbnail-image"
              />
            {:else}
              <div class="thumbnail-placeholder">
                <div class="terminal-id">{id}</div>
              </div>
            {/if}
            
            <!-- Terminal title overlay -->
            <div class="terminal-overlay">
              <div class="terminal-title">
                {terminalTitles[id] || `Terminal ${id}`}
              </div>
            </div>
          </div>
          
          <!-- Status indicator -->
          <div class="status-indicator">
            {#if focusedTerminals.includes(id)}
              <div class="status-dot active"></div>
            {:else}
              <div class="status-dot idle"></div>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  </div>
{/if}

<style lang="postcss">
  .terminals-bar {
    @apply fixed z-40 bg-theme-bg border border-theme-border;
    @apply shadow-lg rounded-lg backdrop-blur-sm;
    @apply transition-all duration-200;
    backdrop-filter: blur(8px);
    background: rgba(var(--color-background), 0.95);
  }
  
  /* Positioning */
  .terminals-bar.position-top {
    @apply left-1/2 transform -translate-x-1/2;
    top: var(--terminals-bar-offset, 1rem);
  }
  
  .terminals-bar.position-bottom {
    @apply left-1/2 transform -translate-x-1/2;
    bottom: var(--terminals-bar-offset, 1rem);
  }
  
  .terminals-bar.position-left {
    @apply top-1/2 transform -translate-y-1/2;
    left: var(--terminals-bar-offset, 1rem);
  }
  
  .terminals-bar.position-right {
    @apply top-1/2 transform -translate-y-1/2;
    right: var(--terminals-bar-offset, 1rem);
  }
  
  /* Layout direction with dynamic sizing */
  .terminals-bar.horizontal {
    @apply flex-row;
    max-width: var(--max-dimension);
  }
  
  .terminals-bar.vertical {
    @apply flex-col;
    max-height: var(--max-dimension);
  }
  
  /* Pinned state */
  .terminals-bar.pinned {
    @apply bg-opacity-100;
  }
  
  /* Header */
  .terminals-bar-header {
    @apply flex items-center justify-between p-2 border-b border-theme-border;
    @apply bg-theme-bg-secondary rounded-t-lg;
  }
  
  .terminals-bar.vertical .terminals-bar-header {
    @apply border-b-0 border-r border-theme-border rounded-tr-none rounded-bl-lg;
  }
  
  .header-info {
    @apply flex items-center gap-1 text-theme-fg-secondary text-xs font-medium;
  }
  
  .terminal-count {
    @apply text-theme-accent font-bold;
  }
  
  .header-controls {
    @apply flex items-center gap-1;
  }
  
  .control-button {
    @apply p-1 rounded text-theme-fg-muted hover:text-theme-fg hover:bg-theme-bg-tertiary;
    @apply transition-colors duration-150;
  }
  
  /* Terminals list - no scrolling, dynamic sizing */
  .terminals-list {
    @apply flex p-2 gap-2;
    /* No overflow scrolling - items will scale instead */
  }
  
  .terminals-bar.horizontal .terminals-list {
    @apply flex-row flex-wrap;
    justify-content: center;
  }
  
  .terminals-bar.vertical .terminals-list {
    @apply flex-col;
    align-items: center;
  }
  
  /* Terminal items */
  .terminal-item {
    @apply relative flex flex-col p-2 rounded-lg;
    @apply cursor-pointer transition-all duration-150;
    @apply hover:bg-theme-bg-secondary/50 border border-transparent;
    @apply focus:outline-none focus:ring-2 focus:ring-theme-accent focus:ring-opacity-50;
  }
  
  .terminals-bar.vertical .terminal-item {
    @apply items-center;
  }
  
  .terminal-item.focused {
    @apply bg-blue-500 bg-opacity-10 border-blue-500 border-opacity-30;
  }
  
  .terminal-item.hovered {
    @apply bg-theme-bg-tertiary;
  }
  
  .terminal-item.add-terminal {
    @apply border-dashed border-theme-border hover:border-theme-accent;
    @apply text-theme-fg-secondary hover:text-theme-accent;
  }
  
  
  /* Terminal thumbnail with dynamic sizing */
  .terminal-thumbnail {
    @apply relative flex-shrink-0 rounded-lg border border-theme-border;
    @apply overflow-hidden bg-black shadow-md;
    width: var(--thumbnail-width);
    height: var(--thumbnail-height);
  }
  
  .thumbnail-image {
    @apply w-full h-full object-contain;
    /* High quality image rendering */
    image-rendering: -webkit-optimize-contrast;
    image-rendering: -moz-crisp-edges;
    image-rendering: crisp-edges;
    image-rendering: optimizeQuality;
    image-rendering: high-quality;
  }
  
  .thumbnail-placeholder {
    @apply w-full h-full flex items-center justify-center;
    @apply bg-theme-bg-tertiary text-theme-fg-muted;
  }
  
  .terminal-id {
    @apply text-xl font-mono font-bold;
  }
  
  /* Terminal overlay */
  .terminal-overlay {
    @apply absolute bottom-0 left-0 right-0 bg-black bg-opacity-75;
    @apply px-2 py-1 backdrop-blur-sm;
  }
  
  .terminal-title {
    @apply text-white text-xs font-mono truncate;
  }
  
  
  /* Status indicator */
  .status-indicator {
    @apply absolute top-1 right-1 flex-shrink-0;
  }
  
  .status-dot {
    @apply w-3 h-3 rounded-full border-2 border-theme-bg shadow-md;
  }
  
  .status-dot.active {
    @apply bg-green-500 border-green-400;
    animation: pulse 2s infinite;
  }
  
  .status-dot.idle {
    @apply bg-gray-500 border-gray-400;
  }
  
  
  /* No scrollbar needed - dynamic sizing handles all terminals */
  
  /* Responsive adjustments - dynamic sizing handles most cases */
  @media (max-width: 768px) {
    .terminals-bar.horizontal {
      max-width: 95vw; /* Slightly more room on mobile */
    }
    
    .terminals-bar.vertical {
      max-height: 95vh; /* Slightly more room on mobile */
    }
    
    /* Ensure minimum readable size on mobile */
    .terminal-thumbnail {
      min-width: 60px;
      min-height: 36px;
    }
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