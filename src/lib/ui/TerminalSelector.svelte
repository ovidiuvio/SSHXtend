<!-- @component Terminal selector overlay for quick navigation -->
<script lang="ts">
  import { createEventDispatcher, onMount, onDestroy } from "svelte";
  import { fade } from "svelte/transition";
  import type { WsWinsize } from "$lib/protocol";
  
  export let shells: [number, WsWinsize][] = [];
  export let focusedTerminals: number[] = [];
  export let terminalTitles: Record<number, string> = {};
  export let terminalThumbnails: Record<number, {small: string | null, large: string | null}> = {};
  
  const dispatch = createEventDispatcher<{
    select: { id: number, winsize: WsWinsize };
    close: void;
  }>();
  
  let selectedIndex = 0;
  let hoveredIndex: number | null = null;
  let previewTerminal: { id: number, winsize: WsWinsize, title: string, thumbnails: {small: string | null, large: string | null} } | null = null;
  
  // Find initially selected terminal (first focused or first in list)
  $: {
    if (focusedTerminals.length > 0) {
      const focusedIndex = shells.findIndex(([id]) => id === focusedTerminals[0]);
      if (focusedIndex >= 0) {
        selectedIndex = focusedIndex;
      }
    }
  }
  
  // Update preview when selection changes
  $: {
    if (shells.length > 0 && (selectedIndex >= 0 && selectedIndex < shells.length)) {
      const [id, winsize] = shells[selectedIndex];
      updatePreview(id, winsize);
    }
  }
  
  function updatePreview(id: number, winsize: WsWinsize) {
    previewTerminal = {
      id,
      winsize,
      title: terminalTitles[id] || `Terminal ${id}`,
      thumbnails: terminalThumbnails[id] || {small: null, large: null}
    };
  }
  
  function handleKeydown(event: KeyboardEvent) {
    // Prevent default for all our handled keys
    const handledKeys = ['Tab', 'ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 
                        'Enter', 'Escape', ' ', '1', '2', '3', '4', '5', '6', '7', 
                        '8', '9', '0', 'q', 'Q', 'w', 'W', 'e', 'E', 'r', 'R'];
    
    if (handledKeys.includes(event.key)) {
      event.preventDefault();
      event.stopPropagation();
    }
    
    // Navigation with Tab
    if (event.key === 'Tab') {
      if (event.shiftKey) {
        selectedIndex = (selectedIndex - 1 + shells.length) % shells.length;
      } else {
        selectedIndex = (selectedIndex + 1) % shells.length;
      }
      return;
    }
    
    // List navigation with arrow keys (vertical)
    switch (event.key) {
      case 'ArrowUp':
        if (selectedIndex > 0) {
          selectedIndex--;
        }
        break;
      case 'ArrowDown':
        if (selectedIndex < shells.length - 1) {
          selectedIndex++;
        }
        break;
      case 'ArrowLeft':
      case 'ArrowRight':
        // No horizontal navigation in list mode
        break;
    }
    
    // Direct jump with number keys
    if (event.key >= '1' && event.key <= '9') {
      const index = parseInt(event.key) - 1;
      if (index < shells.length) {
        selectedIndex = index;
        selectTerminal();
      }
      return;
    }
    
    if (event.key === '0' && shells.length > 9) {
      selectedIndex = 9;
      selectTerminal();
      return;
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
        selectedIndex = index;
        selectTerminal();
      }
      return;
    }
    
    // Selection
    if (event.key === 'Enter' || event.key === ' ') {
      selectTerminal();
      return;
    }
    
    // Cancel
    if (event.key === 'Escape') {
      dispatch('close');
      return;
    }
  }
  
  function selectTerminal(index: number = selectedIndex) {
    if (index >= 0 && index < shells.length) {
      const [id, winsize] = shells[index];
      dispatch('select', { id, winsize });
    }
  }
  
  function handleItemClick(index: number) {
    selectedIndex = index;
    selectTerminal();
  }
  
  function handleMouseEnter(index: number) {
    hoveredIndex = index;
    // Update preview on hover but don't change selection
    const [id, winsize] = shells[index];
    updatePreview(id, winsize);
  }
  
  function handleMouseLeave() {
    hoveredIndex = null;
    // Restore preview to selected item
    if (selectedIndex >= 0 && selectedIndex < shells.length) {
      const [id, winsize] = shells[selectedIndex];
      updatePreview(id, winsize);
    }
  }
  
  // Get keyboard shortcut key for this terminal
  function getShortcutKey(idx: number): string {
    if (idx < 9) return (idx + 1).toString();
    if (idx === 9) return "0";
    const extraKeys = ["Q", "W", "E", "R"];
    if (idx < 14) return extraKeys[idx - 10];
    return "";
  }
  
  function handleBackdropClick(event: MouseEvent) {
    // Only close if clicking directly on backdrop
    if (event.target === event.currentTarget) {
      dispatch('close');
    }
  }
  
  onMount(() => {
    window.addEventListener('keydown', handleKeydown);
    // Focus trap
    document.body.style.overflow = 'hidden';
  });
  
  onDestroy(() => {
    window.removeEventListener('keydown', handleKeydown);
    document.body.style.overflow = '';
  });
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div 
  class="terminal-selector-overlay"
  on:click={handleBackdropClick}
  transition:fade={{ duration: 200 }}
>
  <div class="terminal-selector-container">
    <!-- Header -->
    <div class="selector-header">
      <h2 class="selector-title">Terminal Quick Selector</h2>
      <div class="selector-subtitle">
        {shells.length} terminal{shells.length !== 1 ? 's' : ''} active
      </div>
    </div>
    
    <!-- Two-panel layout -->
    <div class="selector-content">
      <!-- Left panel: Terminal list -->
      <div class="terminal-list">
        {#each shells as [id, winsize], index (id)}
          <!-- svelte-ignore a11y-click-events-have-key-events -->
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <div 
            class="terminal-item"
            class:selected={selectedIndex === index}
            class:hovered={hoveredIndex === index}
            class:focused={focusedTerminals.includes(id)}
            on:click={() => handleItemClick(index)}
            on:mouseenter={() => handleMouseEnter(index)}
            on:mouseleave={handleMouseLeave}
          >
            <!-- Terminal number badge -->
            {#if getShortcutKey(index)}
              <div class="terminal-badge">
                {getShortcutKey(index)}
              </div>
            {/if}
            
            <!-- Small thumbnail -->
            <div class="small-thumbnail">
              {#if terminalThumbnails[id]?.small}
                <img 
                  src={terminalThumbnails[id].small} 
                  alt="Terminal preview" 
                  class="small-thumbnail-image"
                />
              {:else}
                <div class="small-thumbnail-placeholder">
                  <div class="small-placeholder-icon">⌨️</div>
                </div>
              {/if}
            </div>
            
            <!-- Terminal info -->
            <div class="terminal-info">
              <div class="terminal-title">
                {terminalTitles[id] || `Terminal ${id}`}
              </div>
              <div class="terminal-meta">
                {winsize.cols}×{winsize.rows}
              </div>
            </div>
            
            <!-- Status indicator -->
            <div class="terminal-status">
              {#if focusedTerminals.includes(id)}
                <span class="status-dot active" title="Active"></span>
              {:else}
                <span class="status-dot idle" title="Idle"></span>
              {/if}
            </div>
          </div>
        {/each}
      </div>
      
      <!-- Right panel: Large preview -->
      <div class="preview-panel">
        {#if previewTerminal}
          <div class="preview-header">
            <h3 class="preview-title">{previewTerminal.title}</h3>
            <div class="preview-meta">
              Terminal {previewTerminal.id} • {previewTerminal.winsize.cols}×{previewTerminal.winsize.rows}
            </div>
          </div>
          
          <div class="preview-content">
            {#if previewTerminal.thumbnails.large}
              <img 
                src={previewTerminal.thumbnails.large} 
                alt="Terminal preview" 
                class="preview-image"
              />
            {:else}
              <div class="preview-placeholder">
                <div class="placeholder-icon">⌨️</div>
                <div class="placeholder-text">Terminal Preview</div>
              </div>
            {/if}
          </div>
        {:else}
          <div class="preview-empty">
            <div class="empty-icon">👀</div>
            <div class="empty-text">Hover over a terminal to see preview</div>
          </div>
        {/if}
      </div>
    </div>
    
    <!-- Footer with keyboard shortcuts -->
    <div class="selector-footer">
      <div class="shortcut-group">
        <kbd>Tab</kbd>/<kbd>↑↓</kbd> Navigate
      </div>
      <div class="shortcut-group">
        <kbd>Enter</kbd> Select
      </div>
      <div class="shortcut-group">
        <kbd>1-9</kbd> <kbd>0</kbd> <kbd>Q</kbd> <kbd>W</kbd> <kbd>E</kbd> <kbd>R</kbd> Jump
      </div>
      <div class="shortcut-group">
        <kbd>Esc</kbd> Cancel
      </div>
    </div>
  </div>
</div>

<style lang="postcss">
  .terminal-selector-overlay {
    @apply fixed inset-0 z-50;
    @apply bg-black bg-opacity-70 backdrop-blur-sm;
    @apply flex items-center justify-center p-8;
  }
  
  .terminal-selector-container {
    @apply bg-theme-bg rounded-xl border border-theme-border;
    @apply shadow-2xl max-w-6xl w-full max-h-[90vh];
    @apply flex flex-col;
  }
  
  .selector-header {
    @apply p-6 border-b border-theme-border;
  }
  
  .selector-title {
    @apply text-2xl font-semibold text-theme-fg;
  }
  
  .selector-subtitle {
    @apply text-sm text-theme-fg-secondary mt-1;
  }
  
  /* Two-panel layout */
  .selector-content {
    @apply flex flex-1 overflow-hidden;
  }
  
  /* Left panel: Terminal list */
  .terminal-list {
    @apply w-2/5 border-r border-theme-border overflow-y-auto;
    @apply flex flex-col;
  }
  
  .terminal-item {
    @apply relative flex items-center gap-3 p-4 border-b border-theme-border;
    @apply cursor-pointer transition-all duration-150;
    @apply hover:bg-theme-bg-secondary;
  }
  
  .terminal-item.selected {
    @apply bg-blue-500 bg-opacity-10 border-l-4 border-l-blue-500;
  }
  
  .terminal-item.hovered {
    @apply bg-theme-bg-tertiary;
  }
  
  .terminal-item.focused {
    background: rgba(var(--color-success), 0.05);
  }
  
  .terminal-badge {
    @apply flex-shrink-0 bg-blue-500 text-white;
    @apply rounded-full w-6 h-6 flex items-center justify-center;
    @apply font-bold text-xs;
  }
  
  /* Small thumbnails in list */
  .small-thumbnail {
    @apply flex-shrink-0 w-16 h-10 rounded border border-theme-border;
    @apply overflow-hidden bg-black;
  }
  
  .small-thumbnail-image {
    @apply w-full h-full object-cover;
    /* High quality image rendering */
    image-rendering: -webkit-optimize-contrast;
    image-rendering: -moz-crisp-edges;
    image-rendering: crisp-edges;
    image-rendering: optimizeQuality;
    image-rendering: high-quality;
  }
  
  .small-thumbnail-placeholder {
    @apply w-full h-full flex items-center justify-center;
    @apply bg-theme-bg-tertiary;
  }
  
  .small-placeholder-icon {
    @apply text-xs opacity-50;
  }
  
  .terminal-info {
    @apply flex-1 min-w-0;
  }
  
  .terminal-title {
    @apply font-mono text-sm text-theme-fg;
    @apply overflow-hidden text-ellipsis whitespace-nowrap;
  }
  
  .terminal-meta {
    @apply font-mono text-xs text-theme-fg-secondary mt-1;
  }
  
  .terminal-status {
    @apply flex-shrink-0;
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
  
  /* Right panel: Large preview */
  .preview-panel {
    @apply flex-1 flex flex-col p-6;
    @apply bg-theme-bg-secondary;
  }
  
  .preview-header {
    @apply mb-4;
  }
  
  .preview-title {
    @apply text-lg font-semibold text-theme-fg;
    @apply font-mono;
  }
  
  .preview-meta {
    @apply text-sm text-theme-fg-secondary mt-1;
    @apply font-mono;
  }
  
  .preview-content {
    @apply flex-1 flex items-center justify-center;
    @apply rounded-lg border-2 border-dashed border-theme-border;
    @apply overflow-hidden;
  }
  
  .preview-image {
    @apply max-w-full max-h-full object-contain;
    @apply rounded-lg;
    /* High quality image rendering for preview */
    image-rendering: -webkit-optimize-contrast;
    image-rendering: -moz-crisp-edges;
    image-rendering: crisp-edges;
    image-rendering: optimizeQuality;
    image-rendering: high-quality;
  }
  
  .preview-placeholder,
  .preview-empty {
    @apply flex flex-col items-center justify-center;
    @apply text-theme-fg-secondary;
  }
  
  .placeholder-icon,
  .empty-icon {
    @apply text-4xl mb-2;
  }
  
  .placeholder-text,
  .empty-text {
    @apply text-sm font-medium;
  }
  
  /* Responsive design */
  @media (max-width: 1024px) {
    .terminal-list {
      @apply w-1/2;
    }
  }
  
  @media (max-width: 768px) {
    .selector-content {
      @apply flex-col;
    }
    
    .terminal-list {
      @apply w-full max-h-60 border-r-0 border-b;
    }
    
    .preview-panel {
      @apply p-4;
    }
  }
  
  .selector-footer {
    @apply p-4 border-t border-theme-border;
    @apply flex items-center justify-center gap-6 flex-wrap;
  }
  
  .shortcut-group {
    @apply flex items-center gap-2 text-sm text-theme-fg-secondary;
  }
  
  kbd {
    @apply px-2 py-1 border border-theme-border rounded;
    @apply font-mono text-xs text-theme-fg;
    background: rgb(var(--color-background-secondary));
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