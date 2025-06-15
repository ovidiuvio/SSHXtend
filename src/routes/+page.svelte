<script lang="ts">
  import { onMount } from 'svelte';
  import type { SessionInfo } from '$lib/api';
  import { fetchAllSessions, checkDashboardAuth } from '$lib/api';
  import DashboardHeader from '$lib/ui/dashboard/DashboardHeader.svelte';
  import SessionTable from '$lib/ui/dashboard/SessionTable.svelte';
  import AuthPrompt from '$lib/ui/dashboard/AuthPrompt.svelte';
  import logotypeDark from '$lib/assets/logotype-dark.svg';

  let sessions: SessionInfo[] = [];
  let isAuthenticated = false;
  let authError = '';
  let checkingAuth = true;

  async function checkAuthentication() {
    try {
      const authStatus = await checkDashboardAuth();
      
      if (authStatus.required && !authStatus.valid) {
        isAuthenticated = false;
        authError = '';
      } else {
        isAuthenticated = true;
        authError = '';
      }
    } catch (error) {
      console.error('Authentication check failed:', error);
      // On error, assume no auth required
      isAuthenticated = true;
      authError = '';
    } finally {
      checkingAuth = false;
    }
  }

  async function handleAuthentication(key: string) {
    try {
      // Check authentication with the new key
      const authStatus = await checkDashboardAuth();
      
      if (authStatus.valid) {
        isAuthenticated = true;
        authError = '';
      } else {
        authError = 'Invalid dashboard key';
        // Remove invalid key from storage
        if (typeof window !== 'undefined') {
          localStorage.removeItem('sshx-dashboard-key');
        }
      }
    } catch (error) {
      authError = error instanceof Error ? error.message : 'Authentication failed';
      // Remove invalid key from storage
      if (typeof window !== 'undefined') {
        localStorage.removeItem('sshx-dashboard-key');
      }
    }
  }

  async function handleSessionsLoaded(loadedSessions: SessionInfo[]) {
    // For dashboard header stats, we need all sessions, not just the current page
    try {
      const allSessions = await fetchAllSessions();
      sessions = allSessions;
    } catch (error) {
      console.error('Failed to load all sessions for stats:', error);
      // Fallback to the loaded sessions from current page
      sessions = loadedSessions;
    }
  }

  onMount(() => {
    checkAuthentication();
  });
</script>

<svelte:head>
  <title>sshx Dashboard</title>
  <meta name="description" content="Manage and monitor your SSH workspace sessions" />
</svelte:head>

{#if checkingAuth}
  <div class="min-h-screen bg-theme-bg flex items-center justify-center">
    <div class="text-center space-y-4">
      <svg class="animate-spin h-8 w-8 mx-auto text-orange-600 dark:text-orange-400" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
      <p class="text-theme-fg-muted">Checking authentication...</p>
    </div>
  </div>
{:else if !isAuthenticated}
  <AuthPrompt onAuthenticate={handleAuthentication} error={authError} />
{:else}
  <main class="min-h-screen bg-theme-bg">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
      <!-- Page Header -->
      <div class="mb-6">
        <div class="flex items-center gap-3">
          <img src={logotypeDark} alt="sshx" class="h-7" />
        </div>
      </div>

      <!-- Stats Cards -->
      <DashboardHeader {sessions} />
      
      <!-- Sessions Table -->
      <SessionTable {sessions} onSessionsLoaded={handleSessionsLoaded} />
    </div>
  </main>
{/if}

<style>
  :global(body) {
    background-color: var(--theme-bg);
  }
</style>