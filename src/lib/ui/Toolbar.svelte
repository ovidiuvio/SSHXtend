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
  }>();

  $: zoomPercent = Math.round(zoomLevel * 100);
</script>

<div 
  class="panel inline-block"
  class:px-3={position === "top" || position === "bottom"}
  class:py-2={position === "top" || position === "bottom"}
  class:py-3={position === "left" || position === "right"}
  class:px-2={position === "left" || position === "right"}
>
  <div 
    class="select-none"
    class:flex={position === "top" || position === "bottom"}
    class:flex-col={position === "left" || position === "right"}
    class:items-center={true}
  >
    <a href="/" class="flex-shrink-0"
      ><img src={logo} alt="sshx logo" class="h-10" /></a
    >
    <p 
      class="font-medium"
      class:ml-1.5={position === "top" || position === "bottom"}
      class:mr-2={position === "top" || position === "bottom"}
      class:mt-1.5={position === "left" || position === "right"}
      class:mb-2={position === "left" || position === "right"}
    >sshx</p>

    <div 
      class="divider"
      class:v-divider={position === "top" || position === "bottom"}
      class:h-divider={position === "left" || position === "right"}
    />

    <div 
      class="flex"
      class:space-x-1={position === "top" || position === "bottom"}
      class:flex-col={position === "left" || position === "right"}
      class:space-y-1={position === "left" || position === "right"}
    >
      <button
        class="icon-button"
        on:click={() => dispatch("create")}
        disabled={!connected || !hasWriteAccess}
        title={!connected
          ? "Not connected"
          : hasWriteAccess === false // Only show the "No write access" title after confirming read-only mode.
          ? "No write access"
          : "Create new terminal"}
      >
        <PlusCircleIcon strokeWidth={1.5} class="p-0.5" />
      </button>
      <button
        class="icon-button"
        on:click={() => dispatch("autoArrange")}
        disabled={!connected}
        title="Auto-arrange terminals"
      >
        <GridIcon strokeWidth={1.5} class="p-0.5" />
      </button>
      <button class="icon-button" on:click={() => dispatch("chat")}>
        <MessageSquareIcon strokeWidth={1.5} class="p-0.5" />
        {#if newMessages}
          <div class="activity" />
        {/if}
      </button>
      <button class="icon-button" on:click={() => dispatch("settings")}>
        <SettingsIcon strokeWidth={1.5} class="p-0.5" />
      </button>
    </div>

    <div 
      class="divider"
      class:v-divider={position === "top" || position === "bottom"}
      class:h-divider={position === "left" || position === "right"}
    />

    <div 
      class="flex"
      class:space-x-1={position === "top" || position === "bottom"}
      class:flex-col={position === "left" || position === "right"}
      class:space-y-1={position === "left" || position === "right"}
    >
      <button 
        class="icon-button"
        on:click={() => dispatch("zoomOut")}
        title="Zoom out"
        disabled={zoomLevel <= 0.25}
      >
        <ZoomOutIcon strokeWidth={1.5} class="p-0.5" />
      </button>
      <button 
        class="icon-button zoom-level"
        on:click={() => dispatch("zoomReset")}
        title="Reset zoom"
      >
        <span class="text-xs font-medium">{zoomPercent}%</span>
      </button>
      <button 
        class="icon-button"
        on:click={() => dispatch("zoomIn")}
        title="Zoom in"
        disabled={zoomLevel >= 4}
      >
        <ZoomInIcon strokeWidth={1.5} class="p-0.5" />
      </button>
    </div>

    <div 
      class="divider"
      class:v-divider={position === "top" || position === "bottom"}
      class:h-divider={position === "left" || position === "right"}
    />

    <div 
      class="flex"
      class:space-x-1={position === "top" || position === "bottom"}
      class:flex-col={position === "left" || position === "right"}
      class:space-y-1={position === "left" || position === "right"}
    >
      <button 
        class="icon-button network-status"
        class:connected={connected && !exitReason}
        class:error={exitReason !== null}
        class:connecting={!connected && !exitReason}
        on:click={() => dispatch("networkInfo")}
        title={exitReason ? "Connection error" : connected ? "Connected" : "Connecting..."}
      >
        {#if exitReason !== null}
          <AlertCircleIcon strokeWidth={1.5} class="p-0.5" />
        {:else if connected}
          <WifiIcon strokeWidth={1.5} class="p-0.5" />
        {:else}
          <WifiOffIcon strokeWidth={1.5} class="p-0.5" />
        {/if}
      </button>
      {#if connected && hasWriteAccess === false}
        <button 
          class="icon-button read-only-indicator"
          disabled
          title="Read-only mode"
        >
          <EyeIcon strokeWidth={1.5} class="p-0.5" />
        </button>
      {/if}
      <button 
        class="icon-button" 
        on:click={() => dispatch("togglePin")}
        title={pinned ? "Unpin toolbar" : "Pin toolbar"}
      >
        {#if pinned}
          <LockIcon strokeWidth={1.5} class="p-0.5" />
        {:else}
          <UnlockIcon strokeWidth={1.5} class="p-0.5" />
        {/if}
      </button>
    </div>
  </div>
</div>

<style lang="postcss">
  .v-divider {
    @apply h-5 mx-2 border-l-4 border-theme-border;
  }

  .h-divider {
    @apply w-5 my-2 border-t-4 border-theme-border;
  }

  .icon-button {
    @apply relative rounded-md p-1 hover:bg-theme-bg-tertiary active:bg-theme-accent transition-colors;
    @apply disabled:opacity-50 disabled:bg-transparent;
  }

  .activity {
    @apply absolute top-1 right-0.5 text-xs p-[4.5px] bg-theme-error rounded-full;
  }

  .network-status.connected {
    @apply text-theme-success;
  }

  .network-status.error {
    @apply text-theme-error bg-red-500 bg-opacity-20;
  }

  .network-status.connecting {
    @apply text-theme-warning animate-pulse;
  }

  .read-only-indicator {
    @apply text-theme-warning bg-yellow-500 bg-opacity-20;
  }

  .zoom-level {
    @apply min-w-[3rem] px-1;
  }
</style>
