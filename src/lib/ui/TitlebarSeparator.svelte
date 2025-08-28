<script lang="ts">
  import type { TitlebarSeparator } from "$lib/settings";

  export let type: TitlebarSeparator;
  export let color = "#464f60";

  $: separatorClass = {
    none: "hidden",
    line: "border-t border-solid h-px",
    subtle: "border-t border-solid h-px opacity-50",
  }[type];

  $: separatorStyle = type !== "none" ? `border-color: ${color};` : "";
</script>

{#if type !== "none"}
  <div class={separatorClass} style={separatorStyle} />
{/if}