import { persisted } from "svelte-persisted-store";
import themes, { type ThemeName, defaultTheme } from "./ui/themes";
import { derived, type Readable, readable } from "svelte/store";
import { browser } from "$app/environment";

export type UITheme = "light" | "dark" | "auto";
export type ToolbarPosition = "top" | "bottom" | "left" | "right";
export type TerminalsBarPosition = "top" | "bottom" | "left" | "right";

export type AIProvider = "gemini" | "openrouter";

// Default context windows for known models (in tokens)
export const MODEL_CONTEXT_WINDOWS: Record<string, number> = {
  // Gemini models
  "gemini-2.5-flash": 1048576,  // 1M tokens
  "gemini-2.5-pro": 2097152,    // 2M tokens
  "gemini-2.5-flash-lite": 1048576,
  "gemini-2.0-flash": 1048576,
  "gemini-2.0-flash-lite": 1048576,
  "gemini-1.5-flash": 1048576,
  "gemini-1.5-pro": 2097152,
  
  // OpenRouter models (common ones)
  "anthropic/claude-3.5-sonnet": 200000,
  "anthropic/claude-3.5-haiku": 200000,
  "anthropic/claude-3-opus": 200000,
  "openai/gpt-4-turbo-preview": 128000,
  "openai/gpt-4o": 128000,
  "openai/gpt-4o-mini": 128000,
  "google/gemini-pro-1.5": 1048576,
  "meta-llama/llama-3.1-70b-instruct": 131072,
  "deepseek/deepseek-chat": 64000,
  "mistralai/mistral-large": 128000,
  
  // Default fallback
  "default": 32000
};

export type CopyFormat = "html" | "ansi" | "txt" | "markdown";
export type DownloadBehavior = "modal" | "html" | "ansi" | "txt" | "markdown" | "zip";
export type WallpaperFit = "cover" | "contain" | "fill" | "tile" | "center";
export type TitlebarSeparator = "none" | "line" | "subtle";

export type Settings = {
  name: string;
  theme: ThemeName;
  uiTheme: UITheme;
  scrollback: number;
  fontFamily: string;
  fontSize: number;
  fontWeight: number;
  fontWeightBold: number;
  toolbarPosition: ToolbarPosition;
  zoomLevel: number;
  copyOnSelect: boolean;
  middleClickPaste: boolean;
  // Copy button settings
  copyButtonEnabled: boolean;
  copyButtonFormat: CopyFormat;
  // Download button settings
  downloadButtonEnabled: boolean;
  downloadButtonBehavior: DownloadBehavior;
  // Screenshot button setting
  screenshotButtonEnabled: boolean;
  aiEnabled: boolean;
  aiProvider: AIProvider;
  // Gemini settings
  geminiApiKey: string;
  aiModel: string;
  aiModels: string[]; // Custom list of available Gemini models
  // OpenRouter settings
  openRouterApiKey: string;
  openRouterModel: string;
  openRouterModels: string[]; // Custom list of available OpenRouter models
  // Context management
  aiContextLength: number; // Maximum context length in tokens
  aiAutoCompress: boolean; // Auto-compress when reaching 90% of context
  aiMaxResponseTokens: number; // Maximum tokens in AI response
  // Wallpaper settings
  wallpaperEnabled: boolean;
  wallpaperCurrent: string; // wallpaper id
  wallpaperFit: WallpaperFit;
  wallpaperOpacity: number; // 0.1 to 1.0
  // Titlebar settings
  titlebarSeparator: TitlebarSeparator;
  titlebarSeparatorColor: string; // hex color for separator
  titlebarColor: string; // hex color for titlebar background
  titlebarColorEnabled: boolean; // whether to use custom titlebar color
  // Terminals bar settings
  terminalsBarEnabled: boolean;
  terminalsBarPosition: TerminalsBarPosition;
};

const storedSettings = persisted<Partial<Settings>>("sshx-settings-store", {});

/** A persisted store for settings of the current user. */
export const settings: Readable<Settings> = derived(
  storedSettings,
  ($storedSettings) => {
    // Do some validation on all of the stored settings.
    const name = $storedSettings.name ?? "";

    let theme = $storedSettings.theme;
    if (!theme || !Object.hasOwn(themes, theme)) {
      theme = defaultTheme;
    }

    let uiTheme = $storedSettings.uiTheme;
    if (!uiTheme || !["light", "dark", "auto"].includes(uiTheme)) {
      uiTheme = "auto";
    }

    let scrollback = $storedSettings.scrollback;
    if (typeof scrollback !== "number" || scrollback < 0) {
      scrollback = 5000;
    }

    let fontFamily = $storedSettings.fontFamily;
    if (typeof fontFamily !== "string" || fontFamily.trim() === "") {
      fontFamily =
        '"Fira Code VF", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace';
    }

    let fontSize = $storedSettings.fontSize;
    if (typeof fontSize !== "number" || fontSize < 8 || fontSize > 32) {
      fontSize = 14;
    }

    let fontWeight = $storedSettings.fontWeight;
    if (typeof fontWeight !== "number" || fontWeight < 100 || fontWeight > 900) {
      fontWeight = 400;
    }

    let fontWeightBold = $storedSettings.fontWeightBold;
    if (typeof fontWeightBold !== "number" || fontWeightBold < 100 || fontWeightBold > 900) {
      fontWeightBold = 700;
    }

    let toolbarPosition = $storedSettings.toolbarPosition;
    if (!toolbarPosition || !["top", "bottom", "left", "right"].includes(toolbarPosition)) {
      toolbarPosition = "top";
    }

    let zoomLevel = $storedSettings.zoomLevel;
    if (typeof zoomLevel !== "number" || zoomLevel < 0.25 || zoomLevel > 4) {
      zoomLevel = 1;
    }

    let copyOnSelect = $storedSettings.copyOnSelect;
    if (typeof copyOnSelect !== "boolean") {
      copyOnSelect = true;
    }

    let middleClickPaste = $storedSettings.middleClickPaste;
    if (typeof middleClickPaste !== "boolean") {
      middleClickPaste = true;
    }

    let copyButtonEnabled = $storedSettings.copyButtonEnabled;
    if (typeof copyButtonEnabled !== "boolean") {
      copyButtonEnabled = true;
    }

    let copyButtonFormat = $storedSettings.copyButtonFormat;
    if (!copyButtonFormat || !["html", "ansi", "txt", "markdown"].includes(copyButtonFormat)) {
      copyButtonFormat = "ansi";
    }

    let downloadButtonEnabled = $storedSettings.downloadButtonEnabled;
    if (typeof downloadButtonEnabled !== "boolean") {
      downloadButtonEnabled = true;
    }

    let downloadButtonBehavior = $storedSettings.downloadButtonBehavior;
    if (!downloadButtonBehavior || !["modal", "html", "ansi", "txt", "markdown", "zip"].includes(downloadButtonBehavior)) {
      downloadButtonBehavior = "modal";
    }

    let screenshotButtonEnabled = $storedSettings.screenshotButtonEnabled;
    if (typeof screenshotButtonEnabled !== "boolean") {
      screenshotButtonEnabled = true;
    }

    let aiEnabled = $storedSettings.aiEnabled;
    if (typeof aiEnabled !== "boolean") {
      aiEnabled = false;
    }

    let aiProvider = $storedSettings.aiProvider;
    if (!aiProvider || !["gemini", "openrouter"].includes(aiProvider)) {
      aiProvider = "gemini";
    }

    let geminiApiKey = $storedSettings.geminiApiKey;
    if (typeof geminiApiKey !== "string") {
      geminiApiKey = "";
    }

    let aiModel = $storedSettings.aiModel;
    if (typeof aiModel !== "string" || aiModel === "") {
      aiModel = "gemini-2.5-flash";
    }
    
    let aiModels = $storedSettings.aiModels;
    if (!Array.isArray(aiModels) || aiModels.length === 0) {
      // Default list of common Gemini models
      aiModels = [
        "gemini-2.5-flash",
        "gemini-2.5-pro",
        "gemini-2.5-flash-lite",
        "gemini-2.0-flash",
        "gemini-2.0-flash-lite",
      ];
    }

    let openRouterApiKey = $storedSettings.openRouterApiKey;
    if (typeof openRouterApiKey !== "string") {
      openRouterApiKey = "";
    }

    let openRouterModel = $storedSettings.openRouterModel;
    if (typeof openRouterModel !== "string" || openRouterModel === "") {
      openRouterModel = "anthropic/claude-3.5-sonnet";
    }

    let openRouterModels = $storedSettings.openRouterModels;
    if (!Array.isArray(openRouterModels) || openRouterModels.length === 0) {
      // Default list of popular OpenRouter models
      openRouterModels = [
        "anthropic/claude-3.5-sonnet",
        "anthropic/claude-3.5-haiku",
        "openai/gpt-4-turbo-preview",
        "openai/gpt-4o",
        "openai/gpt-4o-mini",
        "google/gemini-pro-1.5",
        "meta-llama/llama-3.1-70b-instruct",
        "deepseek/deepseek-chat",
        "mistralai/mistral-large",
      ];
    }

    // Get the appropriate model based on provider
    const currentModel = aiProvider === 'gemini' ? aiModel : openRouterModel;
    
    // Context length - use model default or user override
    let aiContextLength = $storedSettings.aiContextLength;
    if (typeof aiContextLength !== "number" || aiContextLength <= 0) {
      // Use default for current model
      aiContextLength = MODEL_CONTEXT_WINDOWS[currentModel] || MODEL_CONTEXT_WINDOWS["default"];
    }

    let aiAutoCompress = $storedSettings.aiAutoCompress;
    if (typeof aiAutoCompress !== "boolean") {
      aiAutoCompress = true; // Default to auto-compression enabled
    }

    let aiMaxResponseTokens = $storedSettings.aiMaxResponseTokens;
    if (typeof aiMaxResponseTokens !== "number" || aiMaxResponseTokens <= 0) {
      aiMaxResponseTokens = 4096; // Default to 4K tokens for response
    }

    let wallpaperEnabled = $storedSettings.wallpaperEnabled;
    if (typeof wallpaperEnabled !== "boolean") {
      wallpaperEnabled = true;
    }

    let wallpaperCurrent = $storedSettings.wallpaperCurrent;
    if (typeof wallpaperCurrent !== "string") {
      wallpaperCurrent = "none";
    }

    let wallpaperFit = $storedSettings.wallpaperFit;
    if (!wallpaperFit || !["cover", "contain", "fill", "tile", "center"].includes(wallpaperFit)) {
      wallpaperFit = "cover";
    }

    let wallpaperOpacity = $storedSettings.wallpaperOpacity;
    if (typeof wallpaperOpacity !== "number" || wallpaperOpacity < 0.1 || wallpaperOpacity > 1.0) {
      wallpaperOpacity = 1.0;
    }

    let titlebarSeparator = $storedSettings.titlebarSeparator;
    if (!titlebarSeparator || !["none", "line", "subtle"].includes(titlebarSeparator)) {
      titlebarSeparator = "none";
    }

    let titlebarSeparatorColor = $storedSettings.titlebarSeparatorColor;
    if (typeof titlebarSeparatorColor !== "string" || titlebarSeparatorColor === "") {
      titlebarSeparatorColor = "#464f60"; // Default subtle gray
    }

    let titlebarColor = $storedSettings.titlebarColor;
    if (typeof titlebarColor !== "string" || titlebarColor === "") {
      titlebarColor = "#2d3748"; // Default dark gray
    }

    let titlebarColorEnabled = $storedSettings.titlebarColorEnabled;
    if (typeof titlebarColorEnabled !== "boolean") {
      titlebarColorEnabled = false; // Default disabled (transparent)
    }

    let terminalsBarEnabled = $storedSettings.terminalsBarEnabled;
    if (typeof terminalsBarEnabled !== "boolean") {
      terminalsBarEnabled = false; // Default disabled (auto-hide for performance)
    }

    let terminalsBarPosition = $storedSettings.terminalsBarPosition;
    if (!terminalsBarPosition || !["top", "bottom", "left", "right"].includes(terminalsBarPosition)) {
      terminalsBarPosition = "bottom";
    }


    return {
      name,
      theme,
      uiTheme,
      scrollback,
      fontFamily,
      fontSize,
      fontWeight,
      fontWeightBold,
      toolbarPosition,
      zoomLevel,
      copyOnSelect,
      middleClickPaste,
      copyButtonEnabled,
      copyButtonFormat,
      downloadButtonEnabled,
      downloadButtonBehavior,
      screenshotButtonEnabled,
      aiEnabled,
      aiProvider,
      geminiApiKey,
      aiModel,
      aiModels,
      openRouterApiKey,
      openRouterModel,
      openRouterModels,
      aiContextLength,
      aiAutoCompress,
      aiMaxResponseTokens,
      wallpaperEnabled,
      wallpaperCurrent,
      wallpaperFit,
      wallpaperOpacity,
      titlebarSeparator,
      titlebarSeparatorColor,
      titlebarColor,
      titlebarColorEnabled,
      terminalsBarEnabled,
      terminalsBarPosition,
    };
  },
);

export function updateSettings(values: Partial<Settings>) {
  storedSettings.update((settings) => ({ ...settings, ...values }));
}

/** A store that tracks the system's preferred color scheme */
export const systemTheme = readable<"light" | "dark">(
  "dark",
  (set: (value: "light" | "dark") => void) => {
    if (!browser) return;

    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const updateTheme = () => set(mediaQuery.matches ? "dark" : "light");

    updateTheme();
    mediaQuery.addEventListener("change", updateTheme);

    return () => mediaQuery.removeEventListener("change", updateTheme);
  },
);

/** A derived store that resolves the actual UI theme based on user preference and system settings */
export const actualUITheme = derived(
  [settings, systemTheme],
  ([$settings, $systemTheme]: [Settings, "light" | "dark"]) => {
    if ($settings.uiTheme === "auto") {
      return $systemTheme;
    }
    return $settings.uiTheme;
  },
);

/** Apply the theme to the document */
export function applyTheme(theme: "light" | "dark") {
  if (!browser) return;

  const html = document.documentElement;
  if (theme === "dark") {
    html.classList.add("dark");
  } else {
    html.classList.remove("dark");
  }
}
