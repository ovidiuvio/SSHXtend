const defaultTheme = require("tailwindcss/defaultTheme");

/** @type {import("tailwindcss").Config} */
const config = {
  content: ["./src/**/*.{html,js,svelte,ts}"],

  darkMode: "class",
  theme: {
    extend: {
      fontFamily: {
        sans: ["Inter Variable", ...defaultTheme.fontFamily.sans],
        mono: ["Fira Code VF", ...defaultTheme.fontFamily.mono],
      },
      colors: {
        theme: {
          bg: {
            DEFAULT: "rgb(var(--color-background) / <alpha-value>)",
            secondary: "rgb(var(--color-background-secondary) / <alpha-value>)",
            tertiary: "rgb(var(--color-background-tertiary) / <alpha-value>)",
          },
          fg: {
            DEFAULT: "rgb(var(--color-foreground) / <alpha-value>)",
            secondary: "rgb(var(--color-foreground-secondary) / <alpha-value>)",
            muted: "rgb(var(--color-foreground-muted) / <alpha-value>)",
          },
          border: {
            DEFAULT: "rgb(var(--color-border) / <alpha-value>)",
            secondary: "rgb(var(--color-border-secondary) / <alpha-value>)",
          },
          input: "rgb(var(--color-input) / <alpha-value>)",
          accent: {
            DEFAULT: "rgb(var(--color-accent) / <alpha-value>)",
            hover: "rgb(var(--color-accent-hover) / <alpha-value>)",
          },
          success: "rgb(var(--color-success) / <alpha-value>)",
          warning: "rgb(var(--color-warning) / <alpha-value>)",
          error: "rgb(var(--color-error) / <alpha-value>)",
        },
      },
    },
  },

  plugins: [],
};

module.exports = config;
