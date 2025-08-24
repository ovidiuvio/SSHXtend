<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import {
    MessageSquareIcon,
    PlusCircleIcon,
    SettingsIcon,
    WifiIcon,
    WifiOffIcon,
    AlertCircleIcon,
    LockIcon,
    UnlockIcon,
    EyeIcon,
    ZoomInIcon,
    ZoomOutIcon,
    GridIcon,
    LayersIcon,
  } from "svelte-feather-icons";

  import logo from "$lib/assets/logo.svg";

  export let connected: boolean;
  export let exitReason: string | null = null;
  export let hasWriteAccess: boolean | undefined;
  export let newMessages: boolean;
  export let pinned: boolean = false;
  export let position: "top" | "bottom" | "left" | "right" = "top";
  export let zoomLevel: number = 1;

  const dispatch = createEventDispatcher<{
    create: void;
    chat: void;
    settings: void;
    networkInfo: void;
    togglePin: void;
    zoomIn: void;
    zoomOut: void;
    zoomReset: void;
    autoArrange: void;
    terminalSelector: void;
  }>();

  $: zoomPercent = Math.round(zoomLevel * 100);
</script>

<div 
  class="toolbar-container panel"
  class:horizontal={position === "top" || position === "bottom"}
  class:vertical={position === "left" || position === "right"}
>
  <div class="toolbar-content">
    <!-- Brand section -->
    <div class="brand-section">
      <a href="/" class="brand-link">
        <img src={logo} alt="sshx logo" class="brand-logo" />
        <span class="brand-text">sshx</span>
      </a>
    </div>

    <!-- Main controls group -->
    <div class="control-group">
      <div class="button-cluster">
        <button
          class="toolbar-button primary"
          on:click={() => dispatch("create")}
          disabled={!connected || !hasWriteAccess}
          title={!connected
            ? "Not connected"
            : hasWriteAccess === false
            ? "No write access"
            : "Create new terminal"}
        >
          <PlusCircleIcon strokeWidth={1.5} size="18" />
        </button>
        <button
          class="toolbar-button"
          on:click={() => dispatch("terminalSelector")}
          disabled={!connected}
          title="Quick terminal selector (Ctrl+` or Cmd+`)"
        >
          <LayersIcon strokeWidth={1.5} size="18" />
        </button>
        <button
          class="toolbar-button"
          on:click={() => dispatch("autoArrange")}
          disabled={!connected}
          title="Auto-arrange terminals"
        >
          <GridIcon strokeWidth={1.5} size="18" />
        </button>
      </div>
      
      <div class="button-cluster zoom-cluster">
        <button 
          class="toolbar-button"
          on:click={() => dispatch("zoomOut")}
          title="Zoom out"
          disabled={zoomLevel <= 0.25}
        >
          <ZoomOutIcon strokeWidth={1.5} size="16" />
        </button>
        <button 
          class="toolbar-button zoom-display"
          on:click={() => dispatch("zoomReset")}
          title="Reset zoom"
        >
          <span class="zoom-text">{zoomPercent}%</span>
        </button>
        <button 
          class="toolbar-button"
          on:click={() => dispatch("zoomIn")}
          title="Zoom in"
          disabled={zoomLevel >= 4}
        >
          <ZoomInIcon strokeWidth={1.5} size="16" />
        </button>
      </div>

      <div class="button-cluster">
        <button class="toolbar-button" on:click={() => dispatch("chat")}>
          <MessageSquareIcon strokeWidth={1.5} size="18" />
          {#if newMessages}
            <div class="notification-dot" />
          {/if}
        </button>
        <button class="toolbar-button" on:click={() => dispatch("settings")}>
          <SettingsIcon strokeWidth={1.5} size="18" />
        </button>
      </div>
    </div>

    <!-- Status section -->
    <div class="status-section">
      <button 
        class="toolbar-button status-button"
        class:status-connected={connected && !exitReason}
        class:status-error={exitReason !== null}
        class:status-connecting={!connected && !exitReason}
        on:click={() => dispatch("networkInfo")}
        title={exitReason ? "Connection error" : connected ? "Connected" : "Connecting..."}
      >
        {#if exitReason !== null}
          <AlertCircleIcon strokeWidth={1.5} size="18" />
        {:else if connected}
          <WifiIcon strokeWidth={1.5} size="18" />
        {:else}
          <WifiOffIcon strokeWidth={1.5} size="18" />
        {/if}
      </button>
      {#if connected && hasWriteAccess === false}
        <button 
          class="toolbar-button status-readonly"
          disabled
          title="Read-only mode"
        >
          <EyeIcon strokeWidth={1.5} size="18" />
        </button>
      {/if}
      <button 
        class="toolbar-button" 
        on:click={() => dispatch("togglePin")}
        title={pinned ? "Unpin toolbar" : "Pin toolbar"}
      >
        {#if pinned}
          <LockIcon strokeWidth={1.5} size="18" />
        {:else}
          <UnlockIcon strokeWidth={1.5} size="18" />
        {/if}
      </button>
    </div>
  </div>
</div>

<style lang="postcss">
  /* Container Layout */
  .toolbar-container {
    @apply inline-block;
  }
  
  .toolbar-container.horizontal {
    @apply px-3 py-2;
  }
  
  .toolbar-container.vertical {
    @apply px-2 py-3;
  }
  
  .toolbar-content {
    @apply flex items-center gap-2 select-none;
  }
  
  .toolbar-container.vertical .toolbar-content {
    @apply flex-col gap-1.5;
  }

  /* Brand Section */
  .brand-section {
    @apply flex-shrink-0;
  }
  
  .brand-link {
    @apply flex items-center gap-1.5 text-theme-fg transition-colors hover:text-theme-accent;
  }
  
  .brand-logo {
    @apply h-6 w-6;
  }
  
  .brand-text {
    @apply font-medium text-sm;
  }
  
  .toolbar-container.vertical .brand-link {
    @apply flex-col gap-0.5;
  }
  
  .toolbar-container.vertical .brand-text {
    @apply text-xs;
  }

  /* Control Groups */
  .control-group {
    @apply flex items-center gap-1.5 flex-1;
  }
  
  .toolbar-container.vertical .control-group {
    @apply flex-col gap-1;
  }
  
  .button-cluster {
    @apply flex items-center gap-0.5 bg-theme-bg-tertiary/20 rounded-md p-0.5;
  }
  
  .toolbar-container.vertical .button-cluster {
    @apply flex-col;
  }
  
  .zoom-cluster {
    @apply bg-theme-bg-tertiary/10;
  }

  /* Toolbar Buttons */
  .toolbar-button {
    @apply relative flex items-center justify-center rounded p-1.5 text-theme-fg-muted;
    @apply hover:bg-theme-bg-tertiary hover:text-theme-fg transition-all duration-150;
    @apply active:bg-theme-accent active:text-white active:scale-95;
    @apply disabled:opacity-40 disabled:bg-transparent disabled:cursor-not-allowed disabled:transform-none;
    @apply focus:outline-none focus:ring-1 focus:ring-theme-accent focus:ring-opacity-50;
  }
  
  .toolbar-button.primary {
    @apply bg-theme-accent text-white hover:bg-theme-accent/90;
  }
  
  .toolbar-button.primary:disabled {
    @apply bg-theme-accent/30 text-white/60;
  }

  /* Zoom Display */
  .zoom-display {
    @apply min-w-[2.5rem] px-2 bg-transparent hover:bg-theme-bg-tertiary/50;
  }
  
  .zoom-text {
    @apply text-xs font-medium text-theme-fg;
  }

  /* Status Section */
  .status-section {
    @apply flex items-center gap-0.5 bg-theme-bg-tertiary/10 rounded-md p-0.5 flex-shrink-0;
  }
  
  .toolbar-container.vertical .status-section {
    @apply flex-col;
  }

  /* Status Button States */
  .status-button.status-connected {
    @apply text-theme-success bg-green-500/10 hover:bg-green-500/20;
  }

  .status-button.status-error {
    @apply text-theme-error bg-red-500/10 hover:bg-red-500/20;
  }

  .status-button.status-connecting {
    @apply text-theme-warning bg-yellow-500/10 animate-pulse;
  }

  .status-readonly {
    @apply text-theme-warning bg-yellow-500/10 hover:bg-yellow-500/20;
  }

  /* Notification Dot */
  .notification-dot {
    @apply absolute top-1 right-1 w-2.5 h-2.5 bg-theme-error rounded-full animate-pulse;
  }

  /* Responsive Adjustments */
  @media (max-width: 768px) {
    .toolbar-container.horizontal {
      @apply px-3 py-2;
    }
    
    .toolbar-content {
      @apply gap-3;
    }
    
    .control-group {
      @apply gap-2;
    }
    
    .brand-text {
      @apply text-sm;
    }
    
    .brand-logo {
      @apply h-7 w-7;
    }
  }
</style>
