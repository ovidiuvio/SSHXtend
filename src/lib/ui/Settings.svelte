<script lang="ts">
  import { ChevronDownIcon, MonitorIcon, TypeIcon, CopyIcon, UserIcon, EyeIcon, EyeOffIcon } from "svelte-feather-icons";
  import SparklesIcon from "./icons/SparklesIcon.svelte";

  import { settings, updateSettings, type UITheme, type ToolbarPosition, type AIProvider, type CopyFormat, type DownloadBehavior, MODEL_CONTEXT_WINDOWS } from "$lib/settings";
  import OverlayMenu from "./OverlayMenu.svelte";
  import themes, { type ThemeName } from "./themes";

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
  let inputCopyOnSelect: boolean;
  let inputMiddleClickPaste: boolean;
  let inputCopyButtonEnabled: boolean;
  let inputCopyButtonFormat: CopyFormat;
  let inputDownloadButtonEnabled: boolean;
  let inputDownloadButtonBehavior: DownloadBehavior;
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
  let newModelName = "";
  let showGeminiApiKey = false;
  let showOpenRouterApiKey = false;

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
    inputCopyOnSelect = $settings.copyOnSelect;
    inputMiddleClickPaste = $settings.middleClickPaste;
    inputCopyButtonEnabled = $settings.copyButtonEnabled;
    inputCopyButtonFormat = $settings.copyButtonFormat;
    inputDownloadButtonEnabled = $settings.downloadButtonEnabled;
    inputDownloadButtonBehavior = $settings.downloadButtonBehavior;
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
    {:else if activeTab === "behavior"}
      <!-- Behavior Tab -->
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
</style>