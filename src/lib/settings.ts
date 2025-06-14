import { persisted } from "svelte-persisted-store";
import themes, { type ThemeName, defaultTheme } from "./ui/themes";
import { derived, type Readable, readable } from "svelte/store";
import { browser } from "$app/environment";

export type UITheme = "light" | "dark" | "auto";

export type Settings = {
  name: string;
  theme: ThemeName;
  uiTheme: UITheme;
  scrollback: number;
  fontFamily: string;
  fontSize: number;
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

    return {
      name,
      theme,
      uiTheme,
      scrollback,
      fontFamily,
      fontSize,
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
