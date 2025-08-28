<script lang="ts">
  import { ChevronDownIcon, MonitorIcon, TypeIcon, CopyIcon, UserIcon, EyeIcon, EyeOffIcon, DownloadIcon, UploadIcon } from "svelte-feather-icons";
  import SparklesIcon from "./icons/SparklesIcon.svelte";
  import WallpaperSettings from "./WallpaperSettings.svelte";

  import { settings, updateSettings, type UITheme, type ToolbarPosition, type TerminalsBarPosition, type AIProvider, type CopyFormat, type DownloadBehavior, type TitlebarSeparator, MODEL_CONTEXT_WINDOWS } from "$lib/settings";
  import OverlayMenu from "./OverlayMenu.svelte";
  import themes, { type ThemeName } from "./themes";
  import { profileManager } from "$lib/profileManager";
  import { makeToast } from "$lib/toast";

  export let open: boolean;

  let inputName: string;
  let inputTheme: ThemeName;
  let inputUITheme: UITheme;
  let inputScrollback: number;
  let inputFontFamily: string;
  let inputFontSize: number;
  let inputFontWeight: number;
  let inputFontWeightBold: number;
  let inputToolbarPosition: ToolbarPosition;
  let inputTerminalsBarEnabled: boolean;
  let inputTerminalsBarPosition: TerminalsBarPosition;
  let inputIdleDisconnectEnabled: boolean;
  let inputCopyOnSelect: boolean;
  let inputMiddleClickPaste: boolean;
  let inputCopyButtonEnabled: boolean;
  let inputCopyButtonFormat: CopyFormat;
  let inputDownloadButtonEnabled: boolean;
  let inputDownloadButtonBehavior: DownloadBehavior;
  let inputScreenshotButtonEnabled: boolean;
  let inputAIEnabled: boolean;
  let inputAIProvider: AIProvider;
  let inputGeminiApiKey: string;
  let inputAIModel: string;
  let inputAIModels: string[] = [];
  let inputOpenRouterApiKey: string;
  let inputOpenRouterModel: string;
  let inputOpenRouterModels: string[] = [];
  let inputAIContextLength: number;
  let inputAIAutoCompress: boolean;
  let inputAIMaxResponseTokens: number;
  let inputTitlebarSeparator: TitlebarSeparator;
  let inputTitlebarSeparatorColor: string;
  let inputTitlebarColor: string;
  let inputTitlebarColorEnabled: boolean;
  let newModelName = "";
  let showGeminiApiKey = false;
  let showOpenRouterApiKey = false;

  // Profile import/export state
  let exportingProfile = false;
  let importingProfile = false;
  let importFileInput: HTMLInputElement;

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
    inputFontWeight = $settings.fontWeight;
    inputFontWeightBold = $settings.fontWeightBold;
    inputToolbarPosition = $settings.toolbarPosition;
    inputTerminalsBarEnabled = $settings.terminalsBarEnabled;
    inputTerminalsBarPosition = $settings.terminalsBarPosition;
    inputIdleDisconnectEnabled = $settings.idleDisconnectEnabled;
    inputCopyOnSelect = $settings.copyOnSelect;
    inputMiddleClickPaste = $settings.middleClickPaste;
    inputCopyButtonEnabled = $settings.copyButtonEnabled;
    inputCopyButtonFormat = $settings.copyButtonFormat;
    inputDownloadButtonEnabled = $settings.downloadButtonEnabled;
    inputDownloadButtonBehavior = $settings.downloadButtonBehavior;
    inputScreenshotButtonEnabled = $settings.screenshotButtonEnabled;
    inputAIEnabled = $settings.aiEnabled;
    inputAIProvider = $settings.aiProvider;
    inputGeminiApiKey = $settings.geminiApiKey;
    inputAIModel = $settings.aiModel;
    inputAIModels = $settings.aiModels;
    inputOpenRouterApiKey = $settings.openRouterApiKey;
    inputOpenRouterModel = $settings.openRouterModel;
    inputOpenRouterModels = $settings.openRouterModels;
    inputAIContextLength = $settings.aiContextLength;
    inputAIAutoCompress = $settings.aiAutoCompress;
    inputAIMaxResponseTokens = $settings.aiMaxResponseTokens;
    inputTitlebarSeparator = $settings.titlebarSeparator;
    inputTitlebarSeparatorColor = $settings.titlebarSeparatorColor;
    inputTitlebarColor = $settings.titlebarColor;
    inputTitlebarColorEnabled = $settings.titlebarColorEnabled;
  }

  type Tab = "profile" | "appearance" | "terminal" | "behavior" | "ai";
  let activeTab: Tab = "profile";

  const tabs: { id: Tab; label: string; icon: any }[] = [
    { id: "profile", label: "Profile", icon: UserIcon },
    { id: "appearance", label: "Appearance", icon: MonitorIcon },
    { id: "terminal", label: "Terminal", icon: TypeIcon },
    { id: "behavior", label: "Behavior", icon: CopyIcon },
    { id: "ai", label: "AI", icon: SparklesIcon },
  ];

  // Profile import/export handlers
  async function handleExportProfile() {
    exportingProfile = true;
    try {
      await profileManager.exportProfile();
      makeToast({
        kind: "success",
        message: "Profile exported successfully",
      });
    } catch (error) {
      makeToast({
        kind: "error",
        message: error instanceof Error ? error.message : "Failed to export profile",
      });
    } finally {
      exportingProfile = false;
    }
  }

  function handleImportClick() {
    importFileInput?.click();
  }

  async function handleImportFileSelect(event: Event) {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    
    if (!file) return;

    importingProfile = true;
    
    try {
      const content = await file.text();
      
      const validation = profileManager.validateImportFile(content);
      
      if (!validation.valid) {
        makeToast({
          kind: "error",
          message: `Import failed: ${validation.errors.join(", ")}`,
        });
        return;
      }

      if (validation.warnings.length > 0) {
        makeToast({
          kind: "warning",
          message: `Import completed with warnings: ${validation.warnings.join(", ")}`,
        });
      }

      const importOptions = {
        includeSettings: true,
        includeWallpapers: true,
        overwriteExistingWallpapers: false,
      };

      await profileManager.importProfile(content, importOptions);
      
      makeToast({
        kind: "success",
        message: "Profile imported successfully",
      });
    } catch (error) {
      console.error("Failed to import profile:", error);
      makeToast({
        kind: "error",
        message: error instanceof Error ? error.message : "Failed to import profile",
      });
    } finally {
      importingProfile = false;
      if (input) input.value = "";
    }
  }

</script>

<OverlayMenu
  title="Terminal Settings"
  description="Customize your collaborative terminal."
  showCloseButton
  {open}
  on:close
>
  <!-- Tab Navigation -->
  <div class="flex gap-1 p-1 bg-theme-bg-tertiary/30 rounded-lg mb-6">
    {#each tabs as tab}
      <button
        class="tab-button"
        class:active={activeTab === tab.id}
        on:click={() => (activeTab = tab.id)}
      >
        <svelte:component this={tab.icon} class="w-4 h-4" />
        <span>{tab.label}</span>
      </button>
    {/each}
  </div>

  <!-- Tab Content -->
  <div class="flex flex-col gap-4">
    {#if activeTab === "profile"}
      <!-- Profile Tab -->
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

      <!-- Profile Import/Export Section -->
      <div class="mt-6">
        <div class="mb-4">
          <h3 class="text-lg font-medium text-theme-fg-primary mb-2">Profile Backup</h3>
          <p class="text-sm text-theme-fg-muted">Export your settings and wallpapers to a file, or import from a backup.</p>
        </div>

        <div class="flex gap-2">
          <!-- Export Button -->
          <button
            class="export-import-btn"
            on:click={handleExportProfile}
            disabled={exportingProfile}
          >
            {#if exportingProfile}
              <div class="w-4 h-4 border-2 border-theme-accent border-t-transparent rounded-full animate-spin"></div>
              Exporting...
            {:else}
              <DownloadIcon class="w-4 h-4" />
              Export
            {/if}
          </button>

          <!-- Import Button -->
          <button
            class="export-import-btn"
            on:click={handleImportClick}
            disabled={importingProfile}
          >
            {#if importingProfile}
              <div class="w-4 h-4 border-2 border-theme-accent border-t-transparent rounded-full animate-spin"></div>
              Importing...
            {:else}
              <UploadIcon class="w-4 h-4" />
              Import
            {/if}
          </button>
        </div>

        <!-- Export Info -->
        <div class="mt-4 p-3 bg-theme-bg-tertiary/25 rounded-lg">
          <p class="text-sm text-theme-fg-muted mb-1">
            <strong>Export includes:</strong>
          </p>
          <ul class="text-xs text-theme-fg-muted space-y-0.5 ml-4">
            <li>• All settings (theme, fonts, behavior, AI configuration)</li>
            <li>• Custom wallpapers and wallpaper settings</li>
            <li>• Toolbar and UI customizations</li>
          </ul>
          <p class="text-xs text-theme-fg-muted mt-2 italic">
            API keys are not included for security reasons.
          </p>
        </div>
      </div>
    {:else if activeTab === "appearance"}
      <!-- Appearance Tab -->
      <div class="item">
        <div>
          <p class="item-title">UI Theme</p>
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

      <!-- Terminals Bar Section -->
      <div class="item">
        <div>
          <p class="item-title">Terminals Bar</p>
          <p class="item-subtitle">Quick terminal switcher bar.</p>
          {#if inputTerminalsBarEnabled}
            <div class="mt-2 p-2 bg-yellow-100 dark:bg-yellow-900/20 border border-yellow-300 dark:border-yellow-700 rounded text-sm text-yellow-800 dark:text-yellow-200">
              ⚠️ Performance impact: Recommend using auto-hide mode instead of pin.
            </div>
          {/if}
        </div>
        <div class="flex gap-2 items-center">
          <input
            id="terminals-bar-enabled"
            type="checkbox"
            bind:checked={inputTerminalsBarEnabled}
            on:change={() => updateSettings({ terminalsBarEnabled: inputTerminalsBarEnabled })}
            class="checkbox"
          />
          <label for="terminals-bar-enabled" class="text-sm text-theme-fg">Enabled</label>
        </div>
      </div>

      {#if inputTerminalsBarEnabled}
        <div class="item">
          <div>
            <p class="item-title">Terminals Bar Position</p>
            <p class="item-subtitle">
              Where to position the terminals bar.
              {#if inputTerminalsBarPosition === inputToolbarPosition}
                <span class="text-theme-warning text-xs block mt-1">
                  ⚠️ Will be offset to avoid collision with main toolbar
                </span>
              {/if}
            </p>
          </div>
          <div class="relative">
            <ChevronDownIcon
              class="absolute top-[11px] right-2.5 w-4 h-4 text-theme-fg-muted"
            />
            <select
              class="input-common !pr-5"
              bind:value={inputTerminalsBarPosition}
              on:change={() => updateSettings({ terminalsBarPosition: inputTerminalsBarPosition })}
            >
              <option value="top">Top</option>
              <option value="bottom">Bottom</option>
              <option value="left">Left</option>
              <option value="right">Right</option>
            </select>
          </div>
        </div>

      {/if}
      
      <!-- Wallpaper Settings -->
      <div class="mt-6">
        <WallpaperSettings />
      </div>

    {:else if activeTab === "terminal"}
      <!-- Terminal Tab -->
      <div class="item">
        <div>
          <p class="item-title">Color Palette</p>
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
          <p class="item-title">Font Weight</p>
          <p class="item-subtitle">Weight of normal text (100-900).</p>
        </div>
        <div class="relative">
          <ChevronDownIcon
            class="absolute top-[11px] right-2.5 w-4 h-4 text-theme-fg-muted"
          />
          <select
            class="input-common !pr-5"
            bind:value={inputFontWeight}
            on:change={() => updateSettings({ fontWeight: inputFontWeight })}
          >
            <option value={100}>100 - Thin</option>
            <option value={200}>200 - Extra Light</option>
            <option value={300}>300 - Light</option>
            <option value={400}>400 - Normal</option>
            <option value={500}>500 - Medium</option>
            <option value={600}>600 - Semi Bold</option>
            <option value={700}>700 - Bold</option>
            <option value={800}>800 - Extra Bold</option>
            <option value={900}>900 - Black</option>
          </select>
        </div>
      </div>
      <div class="item">
        <div>
          <p class="item-title">Bold Font Weight</p>
          <p class="item-subtitle">Weight of bold text (100-900).</p>
        </div>
        <div class="relative">
          <ChevronDownIcon
            class="absolute top-[11px] right-2.5 w-4 h-4 text-theme-fg-muted"
          />
          <select
            class="input-common !pr-5"
            bind:value={inputFontWeightBold}
            on:change={() => updateSettings({ fontWeightBold: inputFontWeightBold })}
          >
            <option value={100}>100 - Thin</option>
            <option value={200}>200 - Extra Light</option>
            <option value={300}>300 - Light</option>
            <option value={400}>400 - Normal</option>
            <option value={500}>500 - Medium</option>
            <option value={600}>600 - Semi Bold</option>
            <option value={700}>700 - Bold</option>
            <option value={800}>800 - Extra Bold</option>
            <option value={900}>900 - Black</option>
          </select>
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

      <!-- Titlebar Settings -->
      <div class="mt-6">
        <div class="mb-4">
          <h3 class="text-lg font-medium text-theme-fg-primary mb-2">Titlebar</h3>
          <p class="text-sm text-theme-fg-muted">Customize the terminal titlebar appearance.</p>
        </div>

        <div class="item">
          <div>
            <p class="item-title">Custom Titlebar Color</p>
            <p class="item-subtitle">Enable custom background color for the terminal titlebar.</p>
          </div>
          <div>
            <label class="switch">
              <input
                type="checkbox"
                bind:checked={inputTitlebarColorEnabled}
                on:change={() => updateSettings({ titlebarColorEnabled: inputTitlebarColorEnabled })}
              />
              <span class="slider"></span>
            </label>
          </div>
        </div>

        {#if inputTitlebarColorEnabled}
          <div class="item">
            <div>
              <p class="item-title">Titlebar Background Color</p>
              <p class="item-subtitle">Background color for the terminal titlebar.</p>
            </div>
            <div>
              <input
                type="color"
                class="w-16 h-10 rounded border border-theme-border cursor-pointer"
                bind:value={inputTitlebarColor}
                on:input={() => updateSettings({ titlebarColor: inputTitlebarColor })}
              />
            </div>
          </div>
        {/if}

        <div class="item">
          <div>
            <p class="item-title">Titlebar Separator</p>
            <p class="item-subtitle">Add a separator line between titlebar and terminal content.</p>
          </div>
          <div class="relative">
            <ChevronDownIcon
              class="absolute top-[11px] right-2.5 w-4 h-4 text-theme-fg-muted"
            />
            <select
              class="input-common !pr-5"
              bind:value={inputTitlebarSeparator}
              on:change={() => updateSettings({ titlebarSeparator: inputTitlebarSeparator })}
            >
              <option value="none">None</option>
              <option value="line">Line</option>
              <option value="subtle">Subtle</option>
            </select>
          </div>
        </div>

        {#if inputTitlebarSeparator !== "none"}
          <div class="item">
            <div>
              <p class="item-title">Separator Color</p>
              <p class="item-subtitle">Color of the separator line.</p>
            </div>
            <div>
              <input
                type="color"
                class="w-16 h-10 rounded border border-theme-border cursor-pointer"
                bind:value={inputTitlebarSeparatorColor}
                on:input={() => updateSettings({ titlebarSeparatorColor: inputTitlebarSeparatorColor })}
              />
            </div>
          </div>
        {/if}
      </div>
    {:else if activeTab === "behavior"}
      <!-- Behavior Tab -->
      <div class="item">
        <div>
          <p class="item-title">Auto-Disconnect When Idle</p>
          <p class="item-subtitle">
            Automatically disconnect the WebSocket connection when inactive to save bandwidth. Timeout: <strong>3 minutes</strong> (single user) or <strong>10 minutes</strong> (multiple users). Reconnects automatically when you resume activity.
          </p>
        </div>
        <div>
          <label class="switch">
            <input
              type="checkbox"
              bind:checked={inputIdleDisconnectEnabled}
              on:change={() => updateSettings({ idleDisconnectEnabled: inputIdleDisconnectEnabled })}
            />
            <span class="slider"></span>
          </label>
        </div>
      </div>
      <div class="item">
        <div>
          <p class="item-title">Copy on Select</p>
          <p class="item-subtitle">
            Automatically copy selected text to clipboard when you select text in the terminal.
          </p>
        </div>
        <div>
          <label class="switch">
            <input
              type="checkbox"
              bind:checked={inputCopyOnSelect}
              on:change={() => updateSettings({ copyOnSelect: inputCopyOnSelect })}
            />
            <span class="slider"></span>
          </label>
        </div>
      </div>
      <div class="item">
        <div>
          <p class="item-title">Middle-Click Paste</p>
          <p class="item-subtitle">
            Paste clipboard content when clicking the middle mouse button in the terminal.
          </p>
        </div>
        <div>
          <label class="switch">
            <input
              type="checkbox"
              bind:checked={inputMiddleClickPaste}
              on:change={() => updateSettings({ middleClickPaste: inputMiddleClickPaste })}
            />
            <span class="slider"></span>
          </label>
        </div>
      </div>
      <div class="item">
        <div>
          <p class="item-title">Copy Button</p>
          <p class="item-subtitle">
            Show a copy button in the terminal title bar to copy terminal content to clipboard.
          </p>
        </div>
        <div>
          <label class="switch">
            <input
              type="checkbox"
              bind:checked={inputCopyButtonEnabled}
              on:change={() => updateSettings({ copyButtonEnabled: inputCopyButtonEnabled })}
            />
            <span class="slider"></span>
          </label>
        </div>
      </div>
      {#if inputCopyButtonEnabled}
        <div class="item">
          <div>
            <p class="item-title">Copy Format</p>
            <p class="item-subtitle">
              Choose which format to copy when using the copy button.
            </p>
          </div>
          <div>
            <select
              bind:value={inputCopyButtonFormat}
              on:change={() => updateSettings({ copyButtonFormat: inputCopyButtonFormat })}
              class="text-sm py-2 px-3 bg-theme-bg-secondary border border-theme-border rounded-lg text-theme-fg-primary"
            >
              <option value="ansi">🎨 ANSI (terminal codes)</option>
              <option value="html">🌐 HTML (formatted)</option>
              <option value="txt">📄 Plain Text</option>
              <option value="markdown">📝 Markdown</option>
            </select>
          </div>
        </div>
      {/if}
      <div class="item">
        <div>
          <p class="item-title">Download Button</p>
          <p class="item-subtitle">
            Show a download/export button in the terminal title bar to export terminal content.
          </p>
        </div>
        <div>
          <label class="switch">
            <input
              type="checkbox"
              bind:checked={inputDownloadButtonEnabled}
              on:change={() => updateSettings({ downloadButtonEnabled: inputDownloadButtonEnabled })}
            />
            <span class="slider"></span>
          </label>
        </div>
      </div>
      {#if inputDownloadButtonEnabled}
        <div class="item">
          <div>
            <p class="item-title">Download Behavior</p>
            <p class="item-subtitle">
              Choose what happens when clicking the download button.
            </p>
          </div>
          <div>
            <select
              bind:value={inputDownloadButtonBehavior}
              on:change={() => updateSettings({ downloadButtonBehavior: inputDownloadButtonBehavior })}
              class="text-sm py-2 px-3 bg-theme-bg-secondary border border-theme-border rounded-lg text-theme-fg-primary"
            >
              <option value="modal">📋 Show export modal (choose format)</option>
              <option value="html">🌐 Download HTML directly</option>
              <option value="ansi">🎨 Download ANSI directly</option>
              <option value="txt">📄 Download plain text directly</option>
              <option value="markdown">📝 Download Markdown directly</option>
              <option value="zip">🗜️ Download all formats (ZIP)</option>
            </select>
          </div>
        </div>
      {/if}
      <div class="item">
        <div>
          <p class="item-title">Screenshot Button</p>
          <p class="item-subtitle">
            Show a screenshot button in the terminal title bar to capture high-resolution screenshots.
          </p>
        </div>
        <div>
          <label class="switch">
            <input
              type="checkbox"
              bind:checked={inputScreenshotButtonEnabled}
              on:change={() => updateSettings({ screenshotButtonEnabled: inputScreenshotButtonEnabled })}
            />
            <span class="slider"></span>
          </label>
        </div>
      </div>
    {:else if activeTab === "ai"}
      <!-- AI Tab -->
      <div class="item">
        <div>
          <p class="item-title">Enable AI Assistant</p>
          <p class="item-subtitle">
            Enable AI-powered assistance for terminal output using {inputAIProvider === 'openrouter' ? 'OpenRouter' : 'Google Gemini'}.
          </p>
        </div>
        <div>
          <label class="switch">
            <input
              type="checkbox"
              bind:checked={inputAIEnabled}
              on:change={() => updateSettings({ aiEnabled: inputAIEnabled })}
            />
            <span class="slider"></span>
          </label>
        </div>
      </div>
      
      {#if inputAIEnabled}
        <div class="item">
          <div>
            <p class="item-title">Auto-Compress Conversations</p>
            <p class="item-subtitle">
              Automatically compress conversation history when approaching context limit (90% usage)
            </p>
          </div>
          <div>
            <label class="switch">
              <input
                type="checkbox"
                bind:checked={inputAIAutoCompress}
                on:change={() => updateSettings({ aiAutoCompress: inputAIAutoCompress })}
              />
              <span class="slider"></span>
            </label>
          </div>
        </div>
      
        <div class="item">
          <div>
            <p class="item-title">AI Provider</p>
            <p class="item-subtitle">
              Choose which AI service to use for terminal assistance.
            </p>
          </div>
          <div class="relative">
            <ChevronDownIcon
              class="absolute top-[11px] right-2.5 w-4 h-4 text-theme-fg-muted"
            />
            <select
              class="input-common !pr-5"
              bind:value={inputAIProvider}
              on:change={() => updateSettings({ aiProvider: inputAIProvider })}
            >
              <option value="gemini">Google Gemini</option>
              <option value="openrouter">OpenRouter</option>
            </select>
          </div>
        </div>
        
        {#if inputAIProvider === 'gemini'}
        <div class="item">
          <div>
            <p class="item-title">Gemini API Key</p>
            <p class="item-subtitle">
              Get your API key from <a href="https://makersuite.google.com/app/apikey" target="_blank" rel="noopener" class="text-theme-accent hover:underline">Google AI Studio</a>
            </p>
          </div>
          <div class="relative">
            {#if showGeminiApiKey}
              <input
                type="text"
                class="input-common !pr-10"
                placeholder="Enter your Gemini API key"
                bind:value={inputGeminiApiKey}
                on:input={() => updateSettings({ geminiApiKey: inputGeminiApiKey })}
              />
            {:else}
              <input
                type="password"
                class="input-common !pr-10"
                placeholder="Enter your Gemini API key"
                bind:value={inputGeminiApiKey}
                on:input={() => updateSettings({ geminiApiKey: inputGeminiApiKey })}
              />
            {/if}
            <button
              type="button"
              class="absolute right-2 top-1/2 -translate-y-1/2 p-1 rounded hover:bg-theme-bg-tertiary transition-colors"
              on:click={() => showGeminiApiKey = !showGeminiApiKey}
            >
              {#if showGeminiApiKey}
                <EyeOffIcon class="w-4 h-4 text-theme-fg-muted" />
              {:else}
                <EyeIcon class="w-4 h-4 text-theme-fg-muted" />
              {/if}
            </button>
          </div>
        </div>
        
        <div class="item">
          <div>
            <p class="item-title">AI Model</p>
            <p class="item-subtitle">
              Select or add Gemini models for AI assistance.
            </p>
          </div>
          <div class="space-y-2">
            <div class="relative">
              <ChevronDownIcon
                class="absolute top-[11px] right-2.5 w-4 h-4 text-theme-fg-muted"
              />
              <select
                class="input-common !pr-5"
                bind:value={inputAIModel}
                on:change={() => updateSettings({ aiModel: inputAIModel })}
              >
                {#each inputAIModels as model}
                  <option value={model}>{model}</option>
                {/each}
              </select>
            </div>
            
            <!-- Add custom model -->
            <div class="flex gap-2">
              <input
                type="text"
                class="input-common flex-1"
                placeholder="Add custom model (e.g., gemini-exp-1206)"
                bind:value={newModelName}
                on:keydown={(e) => {
                  if (e.key === 'Enter' && newModelName.trim()) {
                    if (!inputAIModels.includes(newModelName.trim())) {
                      inputAIModels = [...inputAIModels, newModelName.trim()];
                      updateSettings({ aiModels: inputAIModels });
                      if (!inputAIModel) {
                        inputAIModel = newModelName.trim();
                        updateSettings({ aiModel: inputAIModel });
                      }
                    }
                    newModelName = '';
                  }
                }}
              />
              <button
                class="btn-primary px-3 py-1.5"
                on:click={() => {
                  if (newModelName.trim() && !inputAIModels.includes(newModelName.trim())) {
                    inputAIModels = [...inputAIModels, newModelName.trim()];
                    updateSettings({ aiModels: inputAIModels });
                    if (!inputAIModel) {
                      inputAIModel = newModelName.trim();
                      updateSettings({ aiModel: inputAIModel });
                    }
                    newModelName = '';
                  }
                }}
              >
                Add
              </button>
            </div>
            
            <!-- Model list with remove buttons -->
            {#if inputAIModels.length > 1}
              <div class="text-xs text-theme-fg-muted mt-2">
                <p class="mb-1">Available models (click to remove):</p>
                <div class="flex flex-wrap gap-1">
                  {#each inputAIModels as model}
                    <button
                      class="px-2 py-0.5 bg-theme-bg-tertiary/50 rounded hover:bg-red-500/20 transition-colors"
                      title="Click to remove {model}"
                      on:click={() => {
                        if (inputAIModels.length > 1) {
                          inputAIModels = inputAIModels.filter(m => m !== model);
                          updateSettings({ aiModels: inputAIModels });
                          if (inputAIModel === model) {
                            inputAIModel = inputAIModels[0];
                            updateSettings({ aiModel: inputAIModel });
                          }
                        }
                      }}
                    >
                      {model} ×
                    </button>
                  {/each}
                </div>
              </div>
            {/if}
          </div>
        </div>
        
        {:else if inputAIProvider === 'openrouter'}
        <div class="item">
          <div>
            <p class="item-title">OpenRouter API Key</p>
            <p class="item-subtitle">
              Get your API key from <a href="https://openrouter.ai/keys" target="_blank" rel="noopener" class="text-theme-accent hover:underline">OpenRouter</a>
            </p>
          </div>
          <div class="relative">
            {#if showOpenRouterApiKey}
              <input
                type="text"
                class="input-common !pr-10"
                placeholder="Enter your OpenRouter API key"
                bind:value={inputOpenRouterApiKey}
                on:input={() => updateSettings({ openRouterApiKey: inputOpenRouterApiKey })}
              />
            {:else}
              <input
                type="password"
                class="input-common !pr-10"
                placeholder="Enter your OpenRouter API key"
                bind:value={inputOpenRouterApiKey}
                on:input={() => updateSettings({ openRouterApiKey: inputOpenRouterApiKey })}
              />
            {/if}
            <button
              type="button"
              class="absolute right-2 top-1/2 -translate-y-1/2 p-1 rounded hover:bg-theme-bg-tertiary transition-colors"
              on:click={() => showOpenRouterApiKey = !showOpenRouterApiKey}
            >
              {#if showOpenRouterApiKey}
                <EyeOffIcon class="w-4 h-4 text-theme-fg-muted" />
              {:else}
                <EyeIcon class="w-4 h-4 text-theme-fg-muted" />
              {/if}
            </button>
          </div>
        </div>
        
        <div class="item">
          <div>
            <p class="item-title">OpenRouter Model</p>
            <p class="item-subtitle">
              Select or add OpenRouter models for AI assistance.
            </p>
          </div>
          <div class="space-y-2">
            <div class="relative">
              <ChevronDownIcon
                class="absolute top-[11px] right-2.5 w-4 h-4 text-theme-fg-muted"
              />
              <select
                class="input-common !pr-5"
                bind:value={inputOpenRouterModel}
                on:change={() => updateSettings({ openRouterModel: inputOpenRouterModel })}
              >
                {#each inputOpenRouterModels as model}
                  <option value={model}>{model}</option>
                {/each}
              </select>
            </div>
            
            <!-- Add custom model -->
            <div class="flex gap-2">
              <input
                type="text"
                class="input-common flex-1"
                placeholder="Add custom model (e.g., openai/gpt-4-turbo)"
                bind:value={newModelName}
                on:keydown={(e) => {
                  if (e.key === 'Enter' && newModelName.trim()) {
                    if (!inputOpenRouterModels.includes(newModelName.trim())) {
                      inputOpenRouterModels = [...inputOpenRouterModels, newModelName.trim()];
                      updateSettings({ openRouterModels: inputOpenRouterModels });
                      if (!inputOpenRouterModel) {
                        inputOpenRouterModel = newModelName.trim();
                        updateSettings({ openRouterModel: inputOpenRouterModel });
                      }
                    }
                    newModelName = '';
                  }
                }}
              />
              <button
                class="btn-primary px-3 py-1.5"
                on:click={() => {
                  if (newModelName.trim() && !inputOpenRouterModels.includes(newModelName.trim())) {
                    inputOpenRouterModels = [...inputOpenRouterModels, newModelName.trim()];
                    updateSettings({ openRouterModels: inputOpenRouterModels });
                    if (!inputOpenRouterModel) {
                      inputOpenRouterModel = newModelName.trim();
                      updateSettings({ openRouterModel: inputOpenRouterModel });
                    }
                    newModelName = '';
                  }
                }}
              >
                Add
              </button>
            </div>
            
            <!-- Model list with remove buttons -->
            {#if inputOpenRouterModels.length > 1}
              <div class="text-xs text-theme-fg-muted mt-2">
                <p class="mb-1">Available models (click to remove):</p>
                <div class="flex flex-wrap gap-1">
                  {#each inputOpenRouterModels as model}
                    <button
                      class="px-2 py-0.5 bg-theme-bg-tertiary/50 rounded hover:bg-red-500/20 transition-colors"
                      title="Click to remove {model}"
                      on:click={() => {
                        if (inputOpenRouterModels.length > 1) {
                          inputOpenRouterModels = inputOpenRouterModels.filter(m => m !== model);
                          updateSettings({ openRouterModels: inputOpenRouterModels });
                          if (inputOpenRouterModel === model) {
                            inputOpenRouterModel = inputOpenRouterModels[0];
                            updateSettings({ openRouterModel: inputOpenRouterModel });
                          }
                        }
                      }}
                    >
                      {model} ×
                    </button>
                  {/each}
                </div>
              </div>
            {/if}
          </div>
        </div>
        {/if}
        
        <!-- Context Management Settings -->
        <div class="item">
          <div>
            <p class="item-title">Context Window Size</p>
            <p class="item-subtitle">
              Maximum tokens for conversation. Default: {MODEL_CONTEXT_WINDOWS[inputAIProvider === 'gemini' ? inputAIModel : inputOpenRouterModel] || MODEL_CONTEXT_WINDOWS["default"]} tokens
            </p>
          </div>
          <div class="space-y-2">
            <input
              type="number"
              class="input-common"
              placeholder="Use model default"
              bind:value={inputAIContextLength}
              on:input={() => {
                if (inputAIContextLength && inputAIContextLength > 0) {
                  updateSettings({ aiContextLength: inputAIContextLength });
                }
              }}
              min="1000"
              max="2097152"
              step="1000"
            />
            <p class="text-xs text-theme-fg-muted">
              Leave empty to use model's default context window
            </p>
          </div>
        </div>
        
        <div class="item">
          <div>
            <p class="item-title">Max Response Length</p>
            <p class="item-subtitle">
              Maximum tokens for AI responses. Default: 4096 tokens
            </p>
          </div>
          <div class="space-y-2">
            <input
              type="number"
              class="input-common"
              placeholder="4096"
              bind:value={inputAIMaxResponseTokens}
              on:input={() => {
                if (inputAIMaxResponseTokens && inputAIMaxResponseTokens > 0) {
                  updateSettings({ aiMaxResponseTokens: inputAIMaxResponseTokens });
                }
              }}
              min="256"
              max="32768"
              step="256"
            />
            <p class="text-xs text-theme-fg-muted">
              Controls the maximum length of AI responses (256-32768 tokens)
            </p>
          </div>
        </div>
      {/if}
    {/if}
  </div>

  <!-- svelte-ignore missing-declaration -->
  <p class="mt-6 text-sm text-right text-theme-fg-muted">
    <a target="_blank" rel="noreferrer" href="https://github.com/ekzhang/sshx"
      >sshx-server v{__APP_VERSION__}</a
    >
  </p>
</OverlayMenu>

<!-- Hidden File Input for Profile Import -->
<input
  type="file"
  bind:this={importFileInput}
  on:change={handleImportFileSelect}
  accept=".json,application/json"
  class="hidden"
/>



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

  .tab-button {
    @apply flex items-center gap-2 px-4 py-2 text-sm rounded-md;
    @apply text-theme-fg-muted hover:text-theme-fg;
    @apply transition-all duration-200 flex-1;
  }

  .tab-button.active {
    @apply bg-theme-bg text-theme-fg font-medium;
    @apply shadow-sm;
  }

  /* Toggle Switch Styles */
  .switch {
    @apply relative inline-block w-14 h-7;
  }

  .switch input {
    @apply opacity-0 w-0 h-0;
  }

  .slider {
    @apply absolute cursor-pointer inset-0;
    @apply bg-theme-bg-tertiary rounded-full;
    @apply transition-all duration-200;
  }

  .slider:before {
    @apply absolute content-[''] h-5 w-5;
    @apply bg-white rounded-full;
    @apply left-1 bottom-1;
    @apply transition-all duration-200;
  }

  input:checked + .slider {
    @apply bg-theme-accent;
  }

  input:checked + .slider:before {
    @apply translate-x-7;
  }

  input:focus + .slider {
    @apply ring-2 ring-theme-accent/50;
  }

  /* Export/Import Button Styles */
  .export-import-btn {
    @apply flex items-center gap-2 px-3 py-1.5 text-sm rounded;
    @apply bg-theme-bg-secondary hover:bg-theme-bg-tertiary;
    @apply border border-theme-border;
    @apply text-theme-fg hover:text-theme-accent;
    @apply transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
  }
</style>