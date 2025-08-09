<script lang="ts">
  import { ChevronDownIcon } from "svelte-feather-icons";

  import { settings, updateSettings, type UITheme, type ToolbarPosition } from "$lib/settings";
  import OverlayMenu from "./OverlayMenu.svelte";
  import themes, { type ThemeName } from "./themes";

  export let open: boolean;

  let inputName: string;
  let inputTheme: ThemeName;
  let inputUITheme: UITheme;
  let inputScrollback: number;
  let inputFontFamily: string;
  let inputFontSize: number;
  let inputToolbarPosition: ToolbarPosition;

  const fontOptions = [
    {
      name: "Fira Code (default)",
      value:
        '"Fira Code VF", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    },
    {
      name: "System Monospace",
      value:
        'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    },
    {
      name: "SF Mono",
      value:
        '"SF Mono", Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    },
    {
      name: "JetBrains Mono",
      value:
        '"JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    },
    {
      name: "Source Code Pro",
      value:
        '"Source Code Pro", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    },
    {
      name: "Roboto Mono",
      value:
        '"Roboto Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    },
    {
      name: "Cascadia Code",
      value:
        '"Cascadia Code", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    },
    {
      name: "Ubuntu Mono",
      value:
        '"Ubuntu Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    },
  ];

  let initialized = false;
  $: open, (initialized = false);
  $: if (!initialized) {
    initialized = true;
    inputName = $settings.name;
    inputTheme = $settings.theme;
    inputUITheme = $settings.uiTheme;
    inputScrollback = $settings.scrollback;
    inputFontFamily = $settings.fontFamily;
    inputFontSize = $settings.fontSize;
    inputToolbarPosition = $settings.toolbarPosition;
  }
</script>

<OverlayMenu
  title="Terminal Settings"
  description="Customize your collaborative terminal."
  showCloseButton
  {open}
  on:close
>
  <div class="flex flex-col gap-4">
    <div class="item">
      <div>
        <p class="item-title">Name</p>
        <p class="item-subtitle">Choose how you appear to other users.</p>
      </div>
      <div>
        <input
          class="input-common"
          placeholder="Your name"
          bind:value={inputName}
          maxlength="50"
          on:input={() => {
            if (inputName.length >= 2) {
              updateSettings({ name: inputName });
            }
          }}
        />
      </div>
    </div>
    <div class="item">
      <div>
        <p class="item-title">Appearance</p>
        <p class="item-subtitle">Overall theme for the interface.</p>
      </div>
      <div class="relative">
        <ChevronDownIcon
          class="absolute top-[11px] right-2.5 w-4 h-4 text-theme-fg-muted"
        />
        <select
          class="input-common !pr-5"
          bind:value={inputUITheme}
          on:change={() => updateSettings({ uiTheme: inputUITheme })}
        >
          <option value="light">Light</option>
          <option value="dark">Dark</option>
          <option value="auto">Auto</option>
        </select>
      </div>
    </div>
    <div class="item">
      <div>
        <p class="item-title">Color palette</p>
        <p class="item-subtitle">Color theme for text in terminals.</p>
      </div>
      <div class="relative">
        <ChevronDownIcon
          class="absolute top-[11px] right-2.5 w-4 h-4 text-theme-fg-muted"
        />
        <select
          class="input-common !pr-5"
          bind:value={inputTheme}
          on:change={() => updateSettings({ theme: inputTheme })}
        >
          {#each Object.keys(themes) as themeName (themeName)}
            <option value={themeName}>{themeName}</option>
          {/each}
        </select>
      </div>
    </div>
    <div class="item">
      <div>
        <p class="item-title">Font Family</p>
        <p class="item-subtitle">Font family used in terminal windows.</p>
      </div>
      <div class="relative">
        <ChevronDownIcon
          class="absolute top-[11px] right-2.5 w-4 h-4 text-theme-fg-muted"
        />
        <select
          class="input-common !pr-5"
          bind:value={inputFontFamily}
          on:change={() => updateSettings({ fontFamily: inputFontFamily })}
        >
          {#each fontOptions as font}
            <option value={font.value}>{font.name}</option>
          {/each}
        </select>
      </div>
    </div>
    <div class="item">
      <div>
        <p class="item-title">Font Size</p>
        <p class="item-subtitle">Size of text in terminal windows (8-32px).</p>
      </div>
      <div>
        <input
          type="number"
          class="input-common"
          bind:value={inputFontSize}
          on:input={() => {
            if (inputFontSize >= 8 && inputFontSize <= 32) {
              updateSettings({ fontSize: inputFontSize });
            }
          }}
          min="8"
          max="32"
          step="1"
        />
      </div>
    </div>
    <div class="item">
      <div>
        <p class="item-title">Scrollback</p>
        <p class="item-subtitle">
          Lines of previous text displayed in the terminal window.
        </p>
      </div>
      <div>
        <input
          type="number"
          class="input-common"
          bind:value={inputScrollback}
          on:input={() => {
            if (inputScrollback >= 0) {
              updateSettings({ scrollback: inputScrollback });
            }
          }}
          step="100"
        />
      </div>
    </div>
    <div class="item">
      <div>
        <p class="item-title">Toolbar Position</p>
        <p class="item-subtitle">Position of the toolbar on the screen.</p>
      </div>
      <div class="relative">
        <ChevronDownIcon
          class="absolute top-[11px] right-2.5 w-4 h-4 text-theme-fg-muted"
        />
        <select
          class="input-common !pr-5"
          bind:value={inputToolbarPosition}
          on:change={() => updateSettings({ toolbarPosition: inputToolbarPosition })}
        >
          <option value="top">Top</option>
          <option value="bottom">Bottom</option>
          <option value="left">Left</option>
          <option value="right">Right</option>
        </select>
      </div>
    </div>
    <!-- <div class="item">
      <div>
        <p class="item-title">Cursor style</p>
        <p class="item-subtitle">Style of live cursors.</p>
      </div>
      <div class="text-red-500">Coming soon</div>
    </div> -->
  </div>

  <!-- svelte-ignore missing-declaration -->
  <p class="mt-6 text-sm text-right text-theme-fg-muted">
    <a target="_blank" rel="noreferrer" href="https://github.com/ekzhang/sshx"
      >sshx-server v{__APP_VERSION__}</a
    >
  </p>
</OverlayMenu>

<style lang="postcss">
  .item {
    @apply bg-theme-bg-tertiary/25 rounded-lg p-4 flex gap-4 flex-col sm:flex-row items-start;
  }

  .item > div:first-child {
    @apply flex-1;
  }

  .item-title {
    @apply font-medium text-theme-fg mb-1;
  }

  .item-subtitle {
    @apply text-sm text-theme-fg-muted;
  }

  .input-common {
    @apply w-52 px-3 py-2 text-sm rounded-md bg-theme-input hover:bg-theme-bg-tertiary;
    @apply border border-theme-border outline-none focus:ring-2 focus:ring-theme-accent/50;
    @apply appearance-none transition-colors;
  }
</style>
