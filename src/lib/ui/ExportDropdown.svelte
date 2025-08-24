<!-- Compact export dropdown menu -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { DownloadIcon, CheckIcon } from 'svelte-feather-icons';
  import type { ExportFormat, ExportOptions } from '$lib/export';
  
  const dispatch = createEventDispatcher<{
    export: { format: ExportFormat; options: ExportOptions };
    close: void;
  }>();

  export let open = false;
  export let hasSelection = false;
  export let terminalTitle = 'Terminal Session';

  let dropdownEl: HTMLDivElement;
  let selectedFormat: ExportFormat = 'html';

  const exportOptions = {
    selectionOnly: false,
    includeTimestamp: true,
    optimizeForVSCode: true,
    title: terminalTitle
  };

  const formatOptions = [
    {
      format: 'html' as ExportFormat,
      name: 'HTML',
      icon: '🌐',
      description: 'VS Code compatible',
      recommended: true
    },
    {
      format: 'zip' as ExportFormat,
      name: 'All Formats',
      icon: '🗜️',
      description: 'ZIP with all types'
    },
    {
      format: 'ansi' as ExportFormat,
      name: 'ANSI',
      icon: '🎨', 
      description: 'Terminal colors'
    },
    {
      format: 'markdown' as ExportFormat,
      name: 'Markdown',
      icon: '📝',
      description: 'Documentation'
    },
    {
      format: 'txt' as ExportFormat,
      name: 'Text',
      icon: '📄',
      description: 'Plain format'
    }
  ];

  function handleExport(format: ExportFormat) {
    const options = {
      ...exportOptions,
      format,
      selectionOnly: hasSelection && exportOptions.selectionOnly,
      title: terminalTitle
    };
    
    dispatch('export', { format, options });
    open = false;
  }

  function handleClickOutside(event: MouseEvent) {
    if (dropdownEl && !dropdownEl.contains(event.target as Node)) {
      open = false;
    }
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      open = false;
    }
  }

  $: if (open) {
    document.addEventListener('click', handleClickOutside);
    document.addEventListener('keydown', handleKeydown);
  } else {
    document.removeEventListener('click', handleClickOutside);
    document.removeEventListener('keydown', handleKeydown);
  }
</script>

{#if open}
  <div 
    bind:this={dropdownEl}
    class="export-dropdown"
    role="menu"
    tabindex="-1"
  >
    <!-- Header -->
    <div class="export-header">
      <DownloadIcon class="h-3 w-3" />
      <span>Export Terminal</span>
    </div>

    <!-- Format Options -->
    <div class="format-list">
      {#each formatOptions as option (option.format)}
        <button
          class="format-item"
          class:selected={selectedFormat === option.format}
          on:click={() => handleExport(option.format)}
          role="menuitem"
        >
          <div class="format-info">
            <span class="format-icon">{option.icon}</span>
            <div class="format-details">
              <span class="format-name">
                {option.name}
                {#if option.recommended}<span class="recommended">★</span>{/if}
              </span>
              <span class="format-desc">{option.description}</span>
            </div>
          </div>
        </button>
      {/each}
    </div>

    <!-- Options -->
    {#if hasSelection}
      <div class="export-options">
        <label class="option-item">
          <input 
            type="checkbox" 
            bind:checked={exportOptions.selectionOnly}
            class="checkbox"
          />
          <span>Selection only</span>
        </label>
      </div>
    {/if}
  </div>
{/if}

<style lang="postcss">
  .export-dropdown {
    @apply absolute right-0 top-full mt-1 z-50;
    @apply bg-theme-bg border border-theme-border rounded-md shadow-lg;
    @apply min-w-[200px] max-w-[240px];
    @apply py-1;
  }

  .export-header {
    @apply flex items-center gap-2 px-3 py-2 border-b border-theme-border;
    @apply text-xs font-medium text-theme-fg-secondary;
  }

  .format-list {
    @apply py-1;
  }

  .format-item {
    @apply w-full px-3 py-2 text-left transition-colors;
    @apply hover:bg-theme-bg-secondary;
    @apply focus:outline-none focus:bg-theme-bg-secondary;
    border: none;
    background: transparent;
    cursor: pointer;
  }

  .format-info {
    @apply flex items-center gap-2;
  }

  .format-icon {
    @apply text-sm flex-shrink-0;
  }

  .format-details {
    @apply flex flex-col min-w-0;
  }

  .format-name {
    @apply text-sm font-medium text-theme-fg flex items-center gap-1;
  }

  .format-desc {
    @apply text-xs text-theme-fg-muted;
  }

  .recommended {
    @apply text-theme-success text-xs;
  }

  .export-options {
    @apply border-t border-theme-border pt-1 mt-1;
  }

  .option-item {
    @apply flex items-center gap-2 px-3 py-2 cursor-pointer;
    @apply hover:bg-theme-bg-secondary text-sm text-theme-fg;
  }

  .checkbox {
    @apply w-3 h-3 rounded text-theme-accent bg-theme-bg border-theme-border;
    @apply focus:ring-1 focus:ring-theme-accent;
  }
</style>