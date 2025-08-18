<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import type { SessionInfo } from '$lib/api';
  import { fetchAllSessions, checkDashboardStatus } from '$lib/api';
  import DashboardHeader from '$lib/ui/dashboard/DashboardHeader.svelte';
  import SessionTable from '$lib/ui/dashboard/SessionTable.svelte';
  import logotypeDark from '$lib/assets/logotype-dark.svg';

  let sessions: SessionInfo[] = [];
  let isDashboardEnabled = false;
  let checkingStatus = true;
  let dashboardKey = '';

  async function handleSessionsLoaded(loadedSessions: SessionInfo[]) {
    // For dashboard header stats, we need all sessions, not just the current page
    try {
      const allSessions = await fetchAllSessions(dashboardKey);
      sessions = allSessions;
    } catch (error) {
      console.error('Failed to load all sessions for stats:', error);
      // Fallback to the loaded sessions from current page
      sessions = loadedSessions;
    }
  }

  async function checkDashboardEnabled() {
    try {
      const status = await checkDashboardStatus(dashboardKey);
      isDashboardEnabled = status.enabled;
    } catch (error) {
      console.error('Dashboard status check failed:', error);
      isDashboardEnabled = false;
    } finally {
      checkingStatus = false;
    }
  }

  onMount(async () => {
    // Extract dashboard key from URL
    dashboardKey = $page.params.key || '';
    
    await checkDashboardEnabled();
  });
</script>

<svelte:head>
  <title>sshx Dashboard</title>
  <meta name="description" content="Private dashboard for SSH workspace sessions" />
</svelte:head>

{#if checkingStatus}
  <div class="min-h-screen bg-theme-bg flex items-center justify-center">
    <div class="text-center space-y-4">
      <svg class="animate-spin h-8 w-8 mx-auto text-orange-600 dark:text-orange-400" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
      <p class="text-theme-fg-muted">Checking dashboard status...</p>
    </div>
  </div>
{:else if !isDashboardEnabled}
  <div class="min-h-screen bg-theme-bg flex items-center justify-center">
    <div class="text-center space-y-6 max-w-md">
      <img src={logotypeDark} alt="sshx" class="h-12 mx-auto" />
      <div class="space-y-3">
        <h1 class="text-2xl font-bold text-theme-fg">Dashboard Not Found</h1>
        <p class="text-theme-fg-muted">
          This dashboard does not exist or has expired.
        </p>
        <div class="bg-theme-bg-secondary rounded-lg p-4 text-left">
          <p class="text-sm text-theme-fg-muted mb-2">To create a new dashboard, connect with:</p>
          <code class="text-xs bg-theme-bg px-2 py-1 rounded text-orange-600 dark:text-orange-400">
            sshx --dashboard
          </code>
          <p class="text-xs text-theme-fg-muted mt-2">Or join an existing dashboard with <code class="bg-theme-bg px-1 rounded">sshx --dashboard &lt;key&gt;</code></p>
        </div>
      </div>
    </div>
  </div>
{:else}
  <main class="min-h-screen bg-theme-bg">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
      <!-- Page Header -->
      <div class="mb-6">
        <div class="flex items-center gap-3">
          <img src={logotypeDark} alt="sshx" class="h-7" />
          <span class="text-sm text-theme-fg-muted">Private Dashboard</span>
        </div>
      </div>

      <!-- Stats Cards -->
      <DashboardHeader {sessions} />
      
      <!-- Sessions Table -->
      <SessionTable {sessions} {dashboardKey} onSessionsLoaded={handleSessionsLoaded} />
    </div>
  </main>
{/if}

<style>
  :global(body) {
    background-color: var(--theme-bg);
  }
</style>