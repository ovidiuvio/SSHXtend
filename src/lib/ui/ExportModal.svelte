<!-- Refined export modal matching SSHXtend UI style -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import {
    Dialog,
    DialogDescription,
    DialogOverlay,
    DialogTitle,
    Transition,
    TransitionChild,
  } from "@rgossiaux/svelte-headlessui";
  import { XIcon, DownloadIcon } from 'svelte-feather-icons';
  import type { ExportFormat, ExportOptions } from '$lib/export';
  
  const dispatch = createEventDispatcher<{
    export: { format: ExportFormat; options: ExportOptions };
    close: void;
  }>();

  export let open = false;
  export let hasSelection = false;
  export let terminalTitle = 'Terminal Session';

  let selectedFormat: ExportFormat = 'html';
  let exportOptions = {
    selectionOnly: false,
    includeTimestamp: true,
    optimizeForVSCode: true,
    title: terminalTitle
  };

  $: exportOptions.title = terminalTitle;

  const formats = [
    { format: 'html', name: 'HTML', icon: '🌐', desc: 'Perfect for VS Code', rec: true },
    { format: 'zip', name: 'All Formats', icon: '🗜️', desc: 'Complete archive' },
    { format: 'ansi', name: 'ANSI', icon: '🎨', desc: 'Terminal colors' },
    { format: 'markdown', name: 'Markdown', icon: '📝', desc: 'Documentation' },
    { format: 'txt', name: 'Plain Text', icon: '📄', desc: 'Simple format' }
  ];

  function handleExport() {
    dispatch('export', {
      format: selectedFormat,
      options: { ...exportOptions, format: selectedFormat }
    });
    open = false;
  }

  function handleClose() {
    dispatch('close');
    open = false;
  }
</script>

<Transition show={open}>
  <Dialog on:close={handleClose} class="fixed inset-0 z-50 grid place-items-center">
    <DialogOverlay class="fixed -z-10 inset-0 bg-theme-bg/50 backdrop-blur-sm" />

    <TransitionChild
      enter="duration-300 ease-out"
      enterFrom="scale-95 opacity-0"
      enterTo="scale-100 opacity-100"
      leave="duration-75 ease-out"
      leaveFrom="scale-100 opacity-100"
      leaveTo="scale-95 opacity-0"
      class="w-full sm:w-[calc(100%-32px)]"
      style="max-width: 340px"
    >
      <div class="relative bg-theme-bg sm:border border-theme-border px-4 py-4 h-screen sm:h-auto max-h-screen sm:rounded-lg overflow-y-auto">
        
        <!-- Close Button -->
        <button
          class="absolute top-3 right-3 p-1 rounded hover:bg-theme-bg-tertiary active:bg-theme-accent transition-colors"
          aria-label="Close export dialog"
          on:click={handleClose}
        >
          <XIcon class="h-3.5 w-3.5" />
        </button>

        <!-- Header -->
        <div class="mb-4 text-center">
          <DialogTitle class="text-lg font-medium mb-1 flex items-center justify-center gap-2">
            <DownloadIcon class="h-4 w-4" />
            Export Terminal
          </DialogTitle>
          <DialogDescription class="text-theme-fg-muted text-xs">
            Choose format and export options
          </DialogDescription>
        </div>

        <!-- Format Selection -->
        <div class="space-y-2 mb-4">
          {#each formats as format}
            <label 
              class="block relative cursor-pointer"
              class:selected={selectedFormat === format.format}
            >
              <input
                type="radio"
                bind:group={selectedFormat}
                value={format.format}
                class="sr-only"
              />
              <div class="format-option">
                <div class="flex items-center gap-2.5">
                  <span class="text-base">{format.icon}</span>
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-1.5">
                      <span class="font-medium text-sm">{format.name}</span>
                      {#if format.rec}<span class="rec">★</span>{/if}
                    </div>
                    <p class="text-xs text-theme-fg-muted mt-0.5 leading-tight">{format.desc}</p>
                  </div>
                </div>
              </div>
            </label>
          {/each}
        </div>

        <!-- Options -->
        <div class="space-y-2 mb-4">
          {#if hasSelection}
            <label class="flex items-center gap-2.5 cursor-pointer text-sm">
              <input
                type="checkbox"
                bind:checked={exportOptions.selectionOnly}
                class="w-4 h-4 text-theme-accent bg-theme-bg border-theme-border rounded focus:ring-theme-accent focus:ring-2 focus:ring-offset-0"
              />
              <span>Export selection only</span>
            </label>
          {/if}
          
          <label class="flex items-center gap-2.5 cursor-pointer text-sm">
            <input
              type="checkbox"
              bind:checked={exportOptions.includeTimestamp}
              class="w-3.5 h-3.5 text-theme-accent bg-theme-bg border-theme-border rounded focus:ring-theme-accent focus:ring-1 focus:ring-offset-0"
            />
            <span>Include timestamp</span>
          </label>

          <label class="flex items-center gap-2.5 cursor-pointer text-sm">
            <input
              type="checkbox"
              bind:checked={exportOptions.optimizeForVSCode}
              class="w-3.5 h-3.5 text-theme-accent bg-theme-bg border-theme-border rounded focus:ring-theme-accent focus:ring-1 focus:ring-offset-0"
            />
            <span>VS Code optimized</span>
          </label>
        </div>

        <!-- Actions -->
        <div class="flex gap-2">
          <button
            class="flex-1 px-3 py-2 text-sm font-medium bg-theme-bg-secondary text-theme-fg border border-theme-border rounded hover:bg-theme-bg-tertiary hover:border-theme-fg transition-colors focus:outline-none focus:ring-1 focus:ring-theme-border focus:ring-offset-0"
            on:click={handleClose}
          >
            Cancel
          </button>
          <button
            class="flex-1 px-3 py-2 text-sm font-medium bg-theme-accent text-white border border-theme-accent rounded hover:bg-theme-accent-hover hover:border-theme-accent-hover transition-colors focus:outline-none focus:ring-1 focus:ring-theme-accent focus:ring-offset-0 flex items-center justify-center gap-1.5"
            on:click={handleExport}
          >
            <DownloadIcon class="h-3.5 w-3.5" />
            Export
          </button>
        </div>
      </div>
    </TransitionChild>
  </Dialog>
</Transition>

<style lang="postcss">
  .format-option {
    @apply p-2.5 border border-theme-border rounded transition-all;
    @apply hover:border-theme-accent hover:bg-theme-bg-secondary/50;
  }

  .selected .format-option {
    @apply border-theme-accent bg-theme-accent/10;
  }

  .rec {
    @apply text-theme-success text-xs;
  }
</style>