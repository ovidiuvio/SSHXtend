<script lang="ts">
  import { KeyIcon } from 'svelte-feather-icons';
  
  export let onAuthenticate: (key: string) => void;
  export let error: string = '';
  
  let dashboardKey = '';
  let loading = false;
  
  async function handleSubmit() {
    if (!dashboardKey.trim()) return;
    
    loading = true;
    try {
      // Store the key and try authentication
      localStorage.setItem('sshx-dashboard-key', dashboardKey.trim());
      onAuthenticate(dashboardKey.trim());
    } finally {
      loading = false;
    }
  }
  
  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter') {
      handleSubmit();
    }
  }
</script>

<div class="min-h-screen bg-theme-bg flex items-center justify-center p-4">
  <div class="max-w-md w-full space-y-6">
    <!-- Logo -->
    <div class="text-center">
      <div class="inline-flex items-center gap-2 text-2xl font-bold text-theme-fg mb-2">
        <span class="text-orange-600 dark:text-orange-400">ssh</span><span class="text-theme-fg">x</span>
        <span class="text-theme-fg-muted text-lg font-normal">dashboard</span>
      </div>
      <p class="text-theme-fg-muted text-sm">Authentication required</p>
    </div>
    
    <!-- Auth Form -->
    <div class="bg-theme-bg-secondary rounded-lg border border-theme-border p-6 space-y-4">
      <div class="flex items-center gap-3 mb-6">
        <div class="p-2 bg-orange-100 dark:bg-orange-900/30 rounded-lg">
          <KeyIcon size="20" class="text-orange-600 dark:text-orange-400" />
        </div>
        <div>
          <h2 class="text-lg font-semibold text-theme-fg">Dashboard Access</h2>
          <p class="text-sm text-theme-fg-muted">Enter your dashboard key to continue</p>
        </div>
      </div>
      
      <div class="space-y-4">
        <div>
          <label for="dashboard-key" class="block text-sm font-medium text-theme-fg mb-2">
            Dashboard Key
          </label>
          <input
            id="dashboard-key"
            type="password"
            bind:value={dashboardKey}
            on:keydown={handleKeydown}
            placeholder="Enter dashboard key"
            class="w-full px-3 py-2 bg-theme-bg border border-theme-border rounded-lg text-theme-fg placeholder-theme-fg-muted focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
            disabled={loading}
          />
        </div>
        
        {#if error}
          <div class="text-red-500 text-sm bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3">
            {error}
          </div>
        {/if}
        
        <button
          on:click={handleSubmit}
          disabled={!dashboardKey.trim() || loading}
          class="w-full flex items-center justify-center gap-2 px-4 py-2 bg-orange-600 hover:bg-orange-700 disabled:bg-theme-bg-muted disabled:text-theme-fg-muted text-white rounded-lg transition-colors disabled:cursor-not-allowed"
        >
          {#if loading}
            <svg class="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          {:else}
            <KeyIcon size="16" />
          {/if}
          {loading ? 'Authenticating...' : 'Access Dashboard'}
        </button>
      </div>
    </div>
    
    <!-- Help Text -->
    <div class="text-center text-xs text-theme-fg-muted space-y-1">
      <p>The dashboard key is set when starting the sshx server with <code class="bg-theme-bg px-1 py-0.5 rounded text-orange-600 dark:text-orange-400">--dashboard-key</code></p>
      <p>or via the <code class="bg-theme-bg px-1 py-0.5 rounded text-orange-600 dark:text-orange-400">SSHX_DASHBOARD_KEY</code> environment variable</p>
    </div>
  </div>
</div>