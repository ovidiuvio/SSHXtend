<script lang="ts">
  import { onMount } from 'svelte';
  import { RefreshCwIcon, ExternalLinkIcon, LockIcon, TerminalIcon, UsersIcon, SearchIcon } from 'svelte-feather-icons';
  import type { SessionInfo, SessionListResponse, PaginationInfo } from '$lib/api';
  import { fetchSessions, formatLastAccessed } from '$lib/api';
  import SessionRow from './SessionRow.svelte';
  import Pagination from './Pagination.svelte';

  export let sessions: SessionInfo[] = [];
  export let dashboardKey: string;
  export let onSessionsLoaded: (sessions: SessionInfo[]) => void = () => {};

  let loading = true;
  let error = '';
  let refreshing = false;
  let searchQuery = '';
  let currentPage = 1;
  let pageSize = 20;
  let pagination: PaginationInfo | null = null;
  let searchTimeout: ReturnType<typeof setTimeout>;

  async function loadSessions() {
    try {
      loading = true;
      error = '';
      const response = await fetchSessions(dashboardKey, {
        page: currentPage,
        pageSize,
        search: searchQuery || undefined,
        sort: 'lastAccessed',
        order: 'desc'
      });
      sessions = response.sessions;
      pagination = response.pagination;
      onSessionsLoaded(sessions);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load sessions';
      console.error('Failed to load sessions:', err);
    } finally {
      loading = false;
    }
  }

  async function refreshSessions() {
    if (refreshing) return;
    
    try {
      refreshing = true;
      const response = await fetchSessions(dashboardKey, {
        page: currentPage,
        pageSize,
        search: searchQuery || undefined,
        sort: 'lastAccessed',
        order: 'desc'
      });
      sessions = response.sessions;
      pagination = response.pagination;
      onSessionsLoaded(sessions);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to refresh sessions';
      console.error('Failed to refresh sessions:', err);
    } finally {
      refreshing = false;
    }
  }

  function handlePageChange(page: number) {
    currentPage = page;
    loadSessions();
  }

  function handleSearchChange() {
    // Debounce search to avoid too many API calls
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      currentPage = 1; // Reset to first page when searching
      loadSessions();
    }, 300);
  }

  onMount(() => {
    loadSessions();
    
    // Auto-refresh every 10 seconds
    const interval = setInterval(refreshSessions, 10000);
    return () => clearInterval(interval);
  });

  // Trigger search when searchQuery changes
  $: if (searchQuery !== undefined) {
    handleSearchChange();
  }
</script>

<div class="space-y-4">
  <!-- Sessions Section -->
  <div class="bg-theme-bg-secondary rounded-lg border border-theme-border">
    <!-- Header -->
    <div class="px-4 py-3 border-b border-theme-border">
      <div class="flex items-center justify-between">
        <h3 class="text-base font-medium text-theme-fg flex items-center gap-2">
          <TerminalIcon size="16" class="text-orange-600 dark:text-orange-400" />
          Sessions
        </h3>
        <button
          on:click={refreshSessions}
          disabled={refreshing}
          class="flex items-center gap-1.5 px-2.5 py-1.5 text-xs bg-theme-bg hover:bg-theme-bg-muted text-theme-fg rounded transition-colors disabled:opacity-50 border border-theme-border"
        >
          <RefreshCwIcon size="12" class={refreshing ? 'animate-spin' : ''} />
          Refresh
        </button>
      </div>
    </div>

    <!-- Search -->
    <div class="px-4 py-2.5 border-b border-theme-border bg-theme-bg">
      <div class="relative max-w-sm">
        <SearchIcon size="14" class="absolute left-2.5 top-1/2 transform -translate-y-1/2 text-theme-fg-muted" />
        <input
          type="text"
          placeholder="Search sessions by name or user"
          bind:value={searchQuery}
          class="w-full pl-8 pr-3 py-1.5 bg-theme-bg-secondary border border-theme-border rounded text-xs text-theme-fg placeholder-theme-fg-muted focus:outline-none focus:ring-1 focus:ring-orange-500 focus:border-transparent"
        />
      </div>
    </div>

    <!-- Content -->
    {#if loading}
      <div class="flex items-center justify-center py-12">
        <RefreshCwIcon size="24" class="animate-spin text-theme-fg-muted" />
        <span class="ml-2 text-theme-fg-muted">Loading sessions...</span>
      </div>
    {:else if error}
      <div class="text-center py-12">
        <p class="text-red-500 mb-4">{error}</p>
        <button
          on:click={loadSessions}
          class="px-4 py-2 bg-theme-bg hover:bg-theme-bg-muted text-theme-fg border border-theme-border rounded-lg transition-colors"
        >
          Try Again
        </button>
      </div>
    {:else if sessions.length === 0}
      <div class="text-center py-12">
        {#if pagination && pagination.total === 0}
          <TerminalIcon size="48" class="mx-auto text-theme-fg-muted mb-4" />
          <h3 class="text-lg font-medium text-theme-fg mb-2">No Active Sessions</h3>
          <p class="text-theme-fg-muted text-sm">
            Start a new session by running <code class="bg-theme-bg px-1 py-0.5 rounded text-orange-600 dark:text-orange-400">sshx --dashboard</code> on any server
          </p>
        {:else}
          <SearchIcon size="48" class="mx-auto text-theme-fg-muted mb-4" />
          <h3 class="text-lg font-medium text-theme-fg mb-2">No sessions found</h3>
          <p class="text-theme-fg-muted text-sm">
            Try adjusting your search criteria
          </p>
        {/if}
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead>
            <tr class="border-b border-theme-border bg-theme-bg">
              <th class="text-left py-2 px-4 text-xs font-medium text-theme-fg-muted uppercase tracking-wide">
                <div class="flex items-center gap-1.5">
                  <TerminalIcon size="12" class="text-orange-600 dark:text-orange-400" />
                  Name
                </div>
              </th>
              <th class="text-left py-2 px-4 text-xs font-medium text-theme-fg-muted uppercase tracking-wide">
                <div class="flex items-center gap-1">
                  <UsersIcon size="12" />
                  Users
                </div>
              </th>
              <th class="text-left py-2 px-4 text-xs font-medium text-theme-fg-muted uppercase tracking-wide">
                <div class="flex items-center gap-1">
                  <RefreshCwIcon size="12" />
                  Last Active
                </div>
              </th>
              <th class="text-left py-2 px-4 text-xs font-medium text-theme-fg-muted uppercase tracking-wide">
                <div class="flex items-center gap-1">
                  <TerminalIcon size="12" />
                  Terminals
                </div>
              </th>
              <th class="text-right py-2 px-4 text-xs font-medium text-theme-fg-muted uppercase tracking-wide">
                <div class="flex items-center gap-1 justify-end">
                  <ExternalLinkIcon size="12" />
                  Actions
                </div>
              </th>
            </tr>
          </thead>
          <tbody class="bg-theme-bg-secondary divide-y divide-theme-border">
            {#each sessions as session (session.name)}
              <SessionRow {session} />
            {/each}
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      {#if pagination}
        <Pagination {pagination} onPageChange={handlePageChange} />
      {/if}
    {/if}
  </div>
</div>