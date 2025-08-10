<script lang="ts">
  export let size = 24;
  export let strokeWidth = 2;
  let className = "";
  export { className as class };
</script>

<svg
  xmlns="http://www.w3.org/2000/svg"
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width={strokeWidth}
  stroke-linecap="round"
  stroke-linejoin="round"
  class={className}
>
  <!-- Main sparkle -->
  <path d="M12 3l1.912 5.813a2 2 0 0 0 1.275 1.275L21 12l-5.813 1.912a2 2 0 0 0-1.275 1.275L12 21l-1.912-5.813a2 2 0 0 0-1.275-1.275L3 12l5.813-1.912a2 2 0 0 0 1.275-1.275L12 3z" />
  <!-- Plus sign to indicate "new" -->
  <circle cx="19" cy="5" r="3.5" fill="currentColor" opacity="0.15" />
  <path d="M19 3v4M17 5h4" stroke-width={strokeWidth * 1.2} />
</svg>