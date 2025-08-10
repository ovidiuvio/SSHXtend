import { persisted } from "svelte-persisted-store";
import themes, { type ThemeName, defaultTheme } from "./ui/themes";
import { derived, type Readable, readable } from "svelte/store";
import { browser } from "$app/environment";

export type UITheme = "light" | "dark" | "auto";
export type ToolbarPosition = "top" | "bottom" | "left" | "right";

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
