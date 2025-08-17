<!-- @component Terminal selector overlay for quick navigation -->
<script lang="ts">
  import { createEventDispatcher, onMount, onDestroy } from "svelte";
  import { fade } from "svelte/transition";
  import type { WsWinsize } from "$lib/protocol";
  import TerminalCard from "./TerminalCard.svelte";
  
  export let shells: [number, WsWinsize][] = [];
  export let focusedTerminals: number[] = [];
  export let terminalTitles: Record<number, string> = {};
  export let terminalThumbnails: Record<number, string | null> = {};
  
  const dispatch = createEventDispatcher<{
    select: { id: number, winsize: WsWinsize };
    close: void;
  }>();
  
  let selectedIndex = 0;
  let hoveredIndex: number | null = null;
  
  // Find initially selected terminal (first focused or first in list)
  $: {
    if (focusedTerminals.length > 0) {
      const focusedIndex = shells.findIndex(([id]) => id === focusedTerminals[0]);
      if (focusedIndex >= 0) {
        selectedIndex = focusedIndex;
      }
    }
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
    
    // Grid navigation with arrow keys (4 columns)
    const cols = 4;
    const rows = Math.ceil(shells.length / cols);
    const currentRow = Math.floor(selectedIndex / cols);
    const currentCol = selectedIndex % cols;
    
    switch (event.key) {
      case 'ArrowUp':
        if (currentRow > 0) {
          selectedIndex = Math.max(0, selectedIndex - cols);
        }
        break;
      case 'ArrowDown':
        if (currentRow < rows - 1) {
          selectedIndex = Math.min(shells.length - 1, selectedIndex + cols);
        }
        break;
      case 'ArrowLeft':
        if (selectedIndex > 0) {
          selectedIndex--;
        }
        break;
      case 'ArrowRight':
        if (selectedIndex < shells.length - 1) {
          selectedIndex++;
        }
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
  
  function handleCardClick(index: number) {
    selectedIndex = index;
    selectTerminal();
  }
  
  function handleMouseEnter(index: number) {
    hoveredIndex = index;
    selectedIndex = index;
  }
  
  function handleMouseLeave() {
    hoveredIndex = null;
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
    
    <!-- Terminal grid -->
    <div class="selector-grid">
      {#each shells as [id, winsize], index (id)}
        <TerminalCard
          {id}
          {winsize}
          {index}
          title={terminalTitles[id] || `Terminal ${id}`}
          thumbnail={terminalThumbnails[id]}
          selected={selectedIndex === index}
          focused={focusedTerminals.includes(id)}
          on:click={() => handleCardClick(index)}
          on:mouseenter={() => handleMouseEnter(index)}
          on:mouseleave={handleMouseLeave}
        />
      {/each}
      
      <!-- Empty slots for visual balance -->
      {#if shells.length % 4 !== 0}
        {#each Array(4 - (shells.length % 4)) as _, i}
          <div class="empty-slot"></div>
        {/each}
      {/if}
    </div>
    
    <!-- Footer with keyboard shortcuts -->
    <div class="selector-footer">
      <div class="shortcut-group">
        <kbd>Tab</kbd>/<kbd>↑↓←→</kbd> Navigate
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
  
  .selector-grid {
    @apply grid grid-cols-4 gap-4 p-6 overflow-y-auto flex-1;
  }
  
  @media (max-width: 1024px) {
    .selector-grid {
      @apply grid-cols-3;
    }
  }
  
  @media (max-width: 768px) {
    .selector-grid {
      @apply grid-cols-2;
    }
  }
  
  @media (max-width: 480px) {
    .selector-grid {
      @apply grid-cols-1;
    }
  }
  
  .empty-slot {
    @apply invisible;
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
</style>