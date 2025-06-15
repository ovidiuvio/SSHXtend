<script lang="ts">
  import { ExternalLinkIcon, LockIcon, TerminalIcon, UsersIcon, CopyIcon } from 'svelte-feather-icons';
  import type { SessionInfo } from '$lib/api';
  import { formatLastAccessed } from '$lib/api';

  export let session: SessionInfo;

  function openSession() {
    if (session.dashboard?.url) {
      // URLs are stored as relative paths, so they work with any domain
      window.open(session.dashboard.url, '_blank');
    } else {
      // Fallback to session ID only (won't work without encryption key)
      window.open(`/s/${session.name}`, '_blank');
    }
  }

  function openWriteSession() {
    if (session.dashboard?.writeUrl) {
      // URLs are stored as relative paths, so they work with any domain
      window.open(session.dashboard.writeUrl, '_blank');
    }
  }

  function copySessionId() {
    navigator.clipboard.writeText(session.name);
    // Could add toast notification here
  }

  $: statusColor = session.userCount > 0 ? 'text-green-500' : 'text-theme-fg-muted';
  $: lastAccessedText = formatLastAccessed(session.lastAccessed);
  $: isOnline = session.userCount > 0;
</script>

<tr class="hover:bg-theme-bg-muted transition-colors">
  <td class="py-2 px-4">
    <div class="flex items-center gap-2">
      <div class="p-1 bg-orange-100 dark:bg-orange-900/30 rounded">
        <TerminalIcon size="12" class="text-orange-600 dark:text-orange-400" />
      </div>
      <div class="flex flex-col">
        {#if session.dashboard?.displayName}
          <div class="flex items-center gap-1.5">
            <span class="text-xs font-medium text-theme-fg">{session.dashboard.displayName}</span>
            {#if session.hasWritePassword}
              <div class="flex items-center gap-0.5 text-xs bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300 px-1 py-0.5 rounded">
                <LockIcon size="8" />
              </div>
            {/if}
          </div>
          <span class="font-mono text-xs text-theme-fg-muted">{session.name}</span>
        {:else}
          <div class="flex items-center gap-1.5">
            <span class="font-mono text-xs text-theme-fg">{session.name}</span>
            {#if session.hasWritePassword}
              <div class="flex items-center gap-0.5 text-xs bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300 px-1 py-0.5 rounded">
                <LockIcon size="8" />
              </div>
            {/if}
          </div>
        {/if}
        {#if session.users.length > 0}
          <div class="text-xs text-theme-fg-muted">
            Connected: {session.users.join(', ')}
          </div>
        {/if}
      </div>
    </div>
  </td>
  
  <td class="py-2 px-4">
    <div class="flex items-center gap-1">
      <span class="text-xs font-medium text-theme-fg">{session.userCount}</span>
      {#if session.userCount > 0}
        <div class="w-1 h-1 bg-green-500 rounded-full"></div>
      {:else}
        <div class="w-1 h-1 bg-gray-400 rounded-full"></div>
      {/if}
    </div>
  </td>
  
  <td class="py-2 px-4">
    <span class="text-xs text-theme-fg-muted">{lastAccessedText}</span>
  </td>
  
  <td class="py-2 px-4">
    <div class="flex items-center gap-1">
      <div class="p-0.5 bg-orange-100 dark:bg-orange-900/30 rounded">
        <TerminalIcon size="10" class="text-orange-600 dark:text-orange-400" />
      </div>
      <span class="text-xs text-theme-fg-muted">{session.shellCount}</span>
    </div>
  </td>
  
  <td class="py-2 px-4 text-right">
    <div class="flex items-center gap-1 justify-end">
      {#if session.dashboard?.writeUrl}
        <button
          on:click={openWriteSession}
          class="p-1 text-amber-600 dark:text-amber-400 hover:bg-amber-50 dark:hover:bg-amber-900/30 rounded transition-colors"
          title="Open with write access"
        >
          <LockIcon size="12" />
        </button>
      {/if}
      <button
        on:click={openSession}
        class="p-1 text-orange-600 dark:text-orange-400 hover:bg-orange-50 dark:hover:bg-orange-900/30 rounded transition-colors"
        title={session.dashboard?.url ? "Open session" : "Session URL not available"}
        disabled={!session.dashboard?.url}
      >
        <ExternalLinkIcon size="12" />
      </button>
      <button
        on:click={copySessionId}
        class="p-1 text-theme-fg-muted hover:text-theme-fg hover:bg-theme-bg-muted rounded transition-colors"
        title="Copy session ID"
      >
        <CopyIcon size="12" />
      </button>
    </div>
  </td>
</tr>

