<script lang="ts">
  import { TerminalIcon, UsersIcon, ServerIcon, ActivityIcon } from 'svelte-feather-icons';
  import type { SessionInfo } from '$lib/api';

  export let sessions: SessionInfo[] = [];

  $: stats = {
    totalSessions: sessions.length,
    activeSessions: sessions.filter(s => s.userCount > 0).length,
    totalShells: sessions.reduce((sum, s) => sum + s.shellCount, 0),
    totalUsers: sessions.reduce((sum, s) => sum + s.userCount, 0)
  };
</script>

<div class="mb-6 space-y-4">
  <!-- Stats Cards -->
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
    <div class="bg-theme-bg-secondary rounded border border-theme-border p-3">
      <div class="flex items-center gap-2.5">
        <div class="p-1.5 bg-orange-100 dark:bg-orange-900/30 rounded">
          <TerminalIcon size="16" class="text-orange-600 dark:text-orange-400" />
        </div>
        <div>
          <div class="text-xl font-semibold text-theme-fg">{stats.totalSessions}</div>
          <div class="text-xs text-theme-fg-muted">Sessions</div>
        </div>
      </div>
    </div>

    <div class="bg-theme-bg-secondary rounded border border-theme-border p-3">
      <div class="flex items-center gap-2.5">
        <div class="p-1.5 bg-green-100 dark:bg-green-900/30 rounded">
          <ActivityIcon size="16" class="text-green-600 dark:text-green-400" />
        </div>
        <div>
          <div class="text-xl font-semibold text-theme-fg">{stats.activeSessions}</div>
          <div class="text-xs text-theme-fg-muted">Active sessions</div>
        </div>
      </div>
    </div>

    <div class="bg-theme-bg-secondary rounded border border-theme-border p-3">
      <div class="flex items-center gap-2.5">
        <div class="p-1.5 bg-blue-100 dark:bg-blue-900/30 rounded">
          <UsersIcon size="16" class="text-blue-600 dark:text-blue-400" />
        </div>
        <div>
          <div class="text-xl font-semibold text-theme-fg">{stats.totalUsers}</div>
          <div class="text-xs text-theme-fg-muted">Connected users</div>
        </div>
      </div>
    </div>

    <div class="bg-theme-bg-secondary rounded border border-theme-border p-3">
      <div class="flex items-center gap-2.5">
        <div class="p-1.5 bg-purple-100 dark:bg-purple-900/30 rounded">
          <ServerIcon size="16" class="text-purple-600 dark:text-purple-400" />
        </div>
        <div>
          <div class="text-xl font-semibold text-theme-fg">{stats.totalShells}</div>
          <div class="text-xs text-theme-fg-muted">Total shells</div>
        </div>
      </div>
    </div>
  </div>
</div>