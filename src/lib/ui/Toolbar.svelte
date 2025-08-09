<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import {
    MessageSquareIcon,
    PlusCircleIcon,
    SettingsIcon,
    WifiIcon,
    LockIcon,
    UnlockIcon,
  } from "svelte-feather-icons";

  import logo from "$lib/assets/logo.svg";

  export let connected: boolean;
  export let hasWriteAccess: boolean | undefined;
  export let newMessages: boolean;
  export let pinned: boolean = false;
  export let position: "top" | "bottom" | "left" | "right" = "top";

  const dispatch = createEventDispatcher<{
    create: void;
    chat: void;
    settings: void;
    networkInfo: void;
    togglePin: void;
  }>();
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
      <button class="icon-button" on:click={() => dispatch("networkInfo")}>
        <WifiIcon strokeWidth={1.5} class="p-0.5" />
      </button>
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
</style>
