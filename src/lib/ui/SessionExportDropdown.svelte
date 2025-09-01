<!-- Session export dropdown menu - exports all terminals in session -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { DownloadIcon, CheckIcon } from 'svelte-feather-icons';
  import type { ExportFormat, ExportOptions } from '$lib/export';
  
  const dispatch = createEventDispatcher<{
    export: { format: ExportFormat; options: ExportOptions };
    close: void;
  }>();

  export let open = false;
  export let terminalCount = 0;

  let dropdownEl: HTMLDivElement;

  const exportOptions = {
    selectionOnly: false,
    includeTimestamp: true,
    optimizeForVSCode: true,
    title: 'Session Export'
  };

  const formatOptions = [
    {
      format: 'zip' as ExportFormat,
      name: 'All Formats',
      icon: '🗜️',
      description: 'Complete archive with all formats',
      recommended: true
    },
    {
      format: 'html' as ExportFormat,
      name: 'HTML Collection',
      icon: '🌐', 
      description: 'VS Code compatible files'
    },
    {
      format: 'ansi' as ExportFormat,
      name: 'ANSI Collection',
      icon: '🎨',
      description: 'Terminal-native files'
    },
    {
      format: 'markdown' as ExportFormat,
      name: 'Markdown Collection',
      icon: '📝',
      description: 'Documentation files'
    },
    {
      format: 'txt' as ExportFormat,
      name: 'Text Collection',
      icon: '📄',
      description: 'Plain text files'
    }
  ];

  function handleExport(format: ExportFormat) {
    console.log('SessionExportDropdown handleExport called with format:', format);
    const options = {
      ...exportOptions,
      format,
      title: `Session Export - ${terminalCount} terminals`
    };
    
    console.log('Dispatching export event with options:', options);
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
      <span>Export Session</span>
      <span class="terminal-count">{terminalCount} terminals</span>
    </div>

    <!-- Format Options -->
    <div class="format-list">
      {#each formatOptions as option (option.format)}
        <button
          class="format-item"
          on:click={() => handleExport(option.format)}
          role="menuitem"
          disabled={terminalCount === 0}
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

    <!-- Info -->
    <div class="export-info">
      <div class="info-text">
        {#if terminalCount === 0}
          <span class="text-theme-fg-muted">No terminals to export</span>
        {:else if terminalCount === 1}
          <span class="text-theme-fg-muted">Exports 1 terminal in selected format</span>
        {:else}
          <span class="text-theme-fg-muted">Exports all {terminalCount} terminals</span>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style lang="postcss">
  .export-dropdown {
    @apply absolute right-0 top-full mt-1 z-50;
    @apply bg-theme-bg border border-theme-border rounded-md shadow-lg;
    @apply min-w-[220px] max-w-[260px];
    @apply py-1;
  }

  .export-header {
    @apply flex items-center gap-2 px-3 py-2 border-b border-theme-border;
    @apply text-xs font-medium text-theme-fg-secondary;
  }

  .terminal-count {
    @apply ml-auto px-2 py-0.5 bg-theme-accent/10 text-theme-accent rounded-full text-xs;
  }

  .format-list {
    @apply py-1;
  }

  .format-item {
    @apply w-full px-3 py-2 text-left transition-colors;
    @apply hover:bg-theme-bg-secondary;
    @apply focus:outline-none focus:bg-theme-bg-secondary;
    @apply disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-transparent;
    border: none;
    background: transparent;
    cursor: pointer;
  }

  .format-item:disabled {
    cursor: not-allowed;
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

  .export-info {
    @apply border-t border-theme-border pt-2 mt-1;
  }

  .info-text {
    @apply px-3 py-1 text-xs;
  }
</style>