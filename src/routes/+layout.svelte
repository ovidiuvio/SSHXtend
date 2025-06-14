<script lang="ts">
  import "@fontsource-variable/inter";

  import "sshx-xterm/css/xterm.css";
  import "../app.css";

  import ToastContainer from "$lib/ui/ToastContainer.svelte";
  import { actualUITheme, applyTheme } from "$lib/settings";
  import { onMount } from "svelte";

  // Apply theme changes
  $: if ($actualUITheme) {
    applyTheme($actualUITheme);
  }

  // Apply initial theme on mount
  onMount(() => {
    applyTheme($actualUITheme);
  });
</script>

<ToastContainer />

<slot />
