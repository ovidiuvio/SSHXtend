<script lang="ts">
  import { ChevronLeftIcon, ChevronRightIcon } from 'svelte-feather-icons';
  import type { PaginationInfo } from '$lib/api';

  export let pagination: PaginationInfo;
  export let onPageChange: (page: number) => void;

  function goToPage(page: number) {
    if (page >= 1 && page <= pagination.totalPages) {
      onPageChange(page);
    }
  }

  function getVisiblePages(): number[] {
    const current = pagination.page;
    const total = pagination.totalPages;
    const maxVisible = 7;
    
    if (total <= maxVisible) {
      return Array.from({ length: total }, (_, i) => i + 1);
    }
    
    const half = Math.floor(maxVisible / 2);
    let start = Math.max(1, current - half);
    let end = Math.min(total, start + maxVisible - 1);
    
    if (end - start + 1 < maxVisible) {
      start = Math.max(1, end - maxVisible + 1);
    }
    
    return Array.from({ length: end - start + 1 }, (_, i) => start + i);
  }

  $: visiblePages = getVisiblePages();
  $: showFirstEllipsis = visiblePages[0] > 1;
  $: showLastEllipsis = visiblePages[visiblePages.length - 1] < pagination.totalPages;
</script>

{#if pagination.totalPages > 1}
  <div class="flex items-center justify-between px-4 py-3 bg-theme-bg border-t border-theme-border">
    <!-- Info -->
    <div class="flex-1 flex justify-between sm:hidden">
      <button
        on:click={() => goToPage(pagination.page - 1)}
        disabled={!pagination.hasPrevious}
        class="relative inline-flex items-center px-4 py-2 border border-theme-border text-sm font-medium rounded-md text-theme-fg bg-theme-bg hover:bg-theme-bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Previous
      </button>
      <button
        on:click={() => goToPage(pagination.page + 1)}
        disabled={!pagination.hasNext}
        class="ml-3 relative inline-flex items-center px-4 py-2 border border-theme-border text-sm font-medium rounded-md text-theme-fg bg-theme-bg hover:bg-theme-bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Next
      </button>
    </div>
    
    <div class="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
      <div>
        <p class="text-sm text-theme-fg-muted">
          Showing
          <span class="font-medium">{(pagination.page - 1) * pagination.pageSize + 1}</span>
          to
          <span class="font-medium">{Math.min(pagination.page * pagination.pageSize, pagination.total)}</span>
          of
          <span class="font-medium">{pagination.total}</span>
          sessions
        </p>
      </div>
      
      <div>
        <nav class="relative z-0 inline-flex rounded-md shadow-sm -space-x-px" aria-label="Pagination">
          <!-- Previous button -->
          <button
            on:click={() => goToPage(pagination.page - 1)}
            disabled={!pagination.hasPrevious}
            class="relative inline-flex items-center px-2 py-2 rounded-l-md border border-theme-border bg-theme-bg text-sm font-medium text-theme-fg-muted hover:bg-theme-bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <span class="sr-only">Previous</span>
            <ChevronLeftIcon size="16" />
          </button>
          
          <!-- First page -->
          {#if showFirstEllipsis}
            <button
              on:click={() => goToPage(1)}
              class="relative inline-flex items-center px-4 py-2 border border-theme-border bg-theme-bg text-sm font-medium text-theme-fg hover:bg-theme-bg-muted"
            >
              1
            </button>
            <span class="relative inline-flex items-center px-4 py-2 border border-theme-border bg-theme-bg text-sm font-medium text-theme-fg-muted">
              ...
            </span>
          {/if}
          
          <!-- Visible page numbers -->
          {#each visiblePages as page}
            <button
              on:click={() => goToPage(page)}
              class="relative inline-flex items-center px-4 py-2 border text-sm font-medium {page === pagination.page 
                ? 'border-orange-500 bg-orange-50 dark:bg-orange-900/30 text-orange-600 dark:text-orange-400 z-10' 
                : 'border-theme-border bg-theme-bg text-theme-fg hover:bg-theme-bg-muted'}"
            >
              {page}
            </button>
          {/each}
          
          <!-- Last page -->
          {#if showLastEllipsis}
            <span class="relative inline-flex items-center px-4 py-2 border border-theme-border bg-theme-bg text-sm font-medium text-theme-fg-muted">
              ...
            </span>
            <button
              on:click={() => goToPage(pagination.totalPages)}
              class="relative inline-flex items-center px-4 py-2 border border-theme-border bg-theme-bg text-sm font-medium text-theme-fg hover:bg-theme-bg-muted"
            >
              {pagination.totalPages}
            </button>
          {/if}
          
          <!-- Next button -->
          <button
            on:click={() => goToPage(pagination.page + 1)}
            disabled={!pagination.hasNext}
            class="relative inline-flex items-center px-2 py-2 rounded-r-md border border-theme-border bg-theme-bg text-sm font-medium text-theme-fg-muted hover:bg-theme-bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <span class="sr-only">Next</span>
            <ChevronRightIcon size="16" />
          </button>
        </nav>
      </div>
    </div>
  </div>
{/if}