<!-- @component Interactive terminal rendered with xterm.js -->
<script lang="ts" context="module">
  import { makeToast } from "$lib/toast";

  // Deduplicated terminal font loading.
  const waitForFonts = (() => {
    let state: "initial" | "loading" | "loaded" = "initial";
    const waitlist: (() => void)[] = [];

    return async function waitForFonts() {
      if (state === "loaded") return;
      else if (state === "initial") {
        const FontFaceObserver = (await import("fontfaceobserver")).default;
        state = "loading";
        try {
          await new FontFaceObserver("Fira Code VF").load();
        } catch (error) {
          makeToast({
            kind: "error",
            message: "Could not load terminal font.",
          });
        }
        state = "loaded";
        for (const fn of waitlist) fn();
      } else {
        await new Promise<void>((resolve) => {
          if (state === "loaded") resolve();
          else waitlist.push(resolve);
        });
      }
    };
  })();
</script>

<script lang="ts">
  import { browser } from "$app/environment";

  import { createEventDispatcher, onDestroy, onMount } from "svelte";
  import type { Terminal } from "sshx-xterm";
  import { Buffer } from "buffer";
  import { DownloadIcon } from "svelte-feather-icons";

  import themes from "./themes";
  import CircleButton from "./CircleButton.svelte";
  import CircleButtons from "./CircleButtons.svelte";
  import { settings } from "$lib/settings";
  import { TypeAheadAddon } from "$lib/typeahead";

  /** Used to determine Cmd versus Ctrl keyboard shortcuts. */
  const isMac = browser && navigator.platform.startsWith("Mac");

  const dispatch = createEventDispatcher<{
    data: Uint8Array;
    close: void;
    shrink: void;
    expand: void;
    bringToFront: void;
    startMove: MouseEvent;
    focus: void;
    blur: void;
  }>();

  const typeahead = new TypeAheadAddon();

  export let rows: number, cols: number;
  export let write: (data: string) => void; // bound function prop

  export let termEl: HTMLDivElement = null as any; // suppress "missing prop" warning
  let term: Terminal | null = null;

  $: theme = themes[$settings.theme];

  $: if (term) {
    // If the theme changes, update existing terminals' appearance.
    term.options.theme = theme;
    term.options.scrollback = $settings.scrollback;
    term.options.fontFamily = $settings.fontFamily;
    term.options.fontSize = $settings.fontSize;
  }

  let loaded = false;
  let focused = false;
  let currentTitle = "Remote Terminal";

  function downloadTerminalText() {
    if (!term) {
      console.warn("Terminal not available for download");
      return;
    }

    // Try the selection method first as it's more reliable
    try {
      term.selectAll();
      const selectedContent = term.getSelection();
      term.clearSelection();

      if (selectedContent && selectedContent.trim()) {
        downloadContent(selectedContent);
        return;
      }
    } catch (e) {
      console.warn("Selection method failed:", e);
    }

    // Fallback to buffer method
    try {
      const buffer = term.buffer.active;
      const lines: string[] = [];

      // Extract all lines from the terminal buffer
      for (let i = 0; i < buffer.length; i++) {
        const line = buffer.getLine(i);
        if (line) {
          const lineText = line.translateToString(true);
          lines.push(lineText);
        }
      }

      const content = lines.join("\n");

      if (content.trim()) {
        downloadContent(content);
      } else {
        // Try to get whatever is visible as fallback
        const visibleContent =
          term.getSelection() || "No terminal content available";
        downloadContent(visibleContent);
      }
    } catch (e) {
      console.error("Buffer method failed:", e);
      const errorMessage = e instanceof Error ? e.message : String(e);
      downloadContent("Error extracting terminal content: " + errorMessage);
    }
  }

  function downloadContent(content: string) {
    try {
      // Create and trigger download
      const blob = new Blob([content], { type: "text/plain" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `terminal-${currentTitle.replace(
        /[^a-zA-Z0-9]/g,
        "_",
      )}-${new Date().toISOString().slice(0, 19).replace(/:/g, "-")}.txt`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (e) {
      console.error("Download failed:", e);
      const errorMessage = e instanceof Error ? e.message : String(e);
      console.error("Download error details:", errorMessage);
    }
  }

  function handleWheelSkipXTerm(event: WheelEvent) {
    event.preventDefault(); // Stop native macOS Chrome zooming on pinch.

    // We stop the event from propagating to the main `.xterm` terminal element,
    // so the xterm.js's event handlers do not fire and scroll the buffer.
    event.stopPropagation();

    // However, we still want it to propagate upward to our pan/zoom handlers,
    // so we re-dispatch the event higher up, skipping xterm.
    termEl?.dispatchEvent(new WheelEvent(event.type, event));
  }

  function setFocused(isFocused: boolean, cursorLayer: HTMLDivElement) {
    if (isFocused && !focused) {
      focused = isFocused;
      cursorLayer.removeEventListener("wheel", handleWheelSkipXTerm);
      dispatch("focus");
    } else if (!isFocused && focused) {
      focused = isFocused;
      cursorLayer.addEventListener("wheel", handleWheelSkipXTerm);
      dispatch("blur");
    }
  }

  const preloadBuffer: string[] = [];

  write = (data: string) => {
    if (!term) {
      // Before the terminal is loaded, push data into a buffer.
      preloadBuffer.push(data);
    } else {
      if (data) data = typeahead.onBeforeProcessData(data);
      term.write(data);
    }
  };

  $: term?.resize(cols, rows);

  onMount(async () => {
    const [{ Terminal }, { WebLinksAddon }, { WebglAddon }, { ImageAddon }] =
      await Promise.all([
        import("sshx-xterm"),
        import("xterm-addon-web-links"),
        import("xterm-addon-webgl"), // NOTE: WebGL contexts are limited by browsers (~16 max)
        import("xterm-addon-image"),
      ]);

    await waitForFonts();

    term = new Terminal({
      allowTransparency: false,
      cursorBlink: false,
      cursorStyle: "block",
      fontFamily: $settings.fontFamily,
      fontSize: $settings.fontSize,
      fontWeight: 400,
      fontWeightBold: 500,
      lineHeight: 1.06,
      scrollback: $settings.scrollback,
      theme,
    });

    // Keyboard shortcuts for natural text editing.
    term.attachCustomKeyEventHandler((event) => {
      if (
        (isMac && event.metaKey && !event.ctrlKey && !event.altKey) ||
        (!isMac && !event.metaKey && event.ctrlKey && !event.altKey)
      ) {
        if (event.key === "ArrowLeft") {
          dispatch("data", new Uint8Array([0x01]));
          return false;
        } else if (event.key === "ArrowRight") {
          dispatch("data", new Uint8Array([0x05]));
          return false;
        } else if (event.key === "Backspace") {
          dispatch("data", new Uint8Array([0x15]));
          return false;
        }
      }
      return true;
    });

    term.loadAddon(new WebLinksAddon());
    // WebGL addon provides GPU acceleration but browsers limit WebGL contexts
    // to ~16 concurrent contexts. Beyond this limit, older contexts are destroyed,
    // causing terminals to become unrenderable (upset emoticon).
    // The 14-terminal limit in Session.svelte prevents this issue.
    term.loadAddon(new WebglAddon());
    term.loadAddon(new ImageAddon({ enableSizeReports: false }));

    term.open(termEl);

    term.resize(cols, rows);
    term.onTitleChange((title) => {
      currentTitle = title;
    });

    // Hack: We artificially disable scrolling when the terminal is not focused.
    // ("termEl" > div.terminal.xterm > div.xterm-screen)
    const screenEl = termEl.querySelector(".xterm-screen")! as HTMLDivElement;
    screenEl.addEventListener("wheel", handleWheelSkipXTerm);

    const focusObserver = new MutationObserver((mutations) => {
      for (const mutation of mutations) {
        if (
          mutation.type === "attributes" &&
          mutation.attributeName === "class"
        ) {
          // The "focus" class is set directly by xterm.js, but there isn't any way to listen for it.
          const target = mutation.target as HTMLElement;
          const isFocused = target.classList.contains("focus");
          setFocused(isFocused, screenEl);
        }
      }
    });
    focusObserver.observe(term.element!, { attributeFilter: ["class"] });

    loaded = true;
    for (const data of preloadBuffer) {
      term.write(data);
    }

    typeahead.reset();
    term.loadAddon(typeahead);

    const utf8 = new TextEncoder();
    term.onData((data: string) => {
      dispatch("data", utf8.encode(data));
    });
    term.onBinary((data: string) => {
      dispatch("data", Buffer.from(data, "binary"));
    });
  });

  onDestroy(() => term?.dispose());
</script>

<div
  class="term-container"
  class:focused
  style:background={theme.background}
  on:mousedown={() => dispatch("bringToFront")}
  on:pointerdown={(event) => event.stopPropagation()}
>
  <div
    class="flex select-none"
    on:mousedown={(event) => dispatch("startMove", event)}
  >
    <div class="flex-1 flex items-center px-3">
      <CircleButtons>
        <!--
          TODO: This should be on:click, but that is not working due to the
          containing element's on:pointerdown `stopPropagation()` call.
        -->
        <CircleButton
          kind="red"
          on:mousedown={(event) => event.button === 0 && dispatch("close")}
        />
        <CircleButton
          kind="yellow"
          on:mousedown={(event) => event.button === 0 && dispatch("shrink")}
        />
        <CircleButton
          kind="green"
          on:mousedown={(event) => event.button === 0 && dispatch("expand")}
        />
      </CircleButtons>
    </div>
    <div
      class="p-2 text-sm text-theme-fg-secondary text-center font-medium overflow-hidden whitespace-nowrap text-ellipsis w-0 flex-grow-[4]"
    >
      {currentTitle}
    </div>
    <div class="flex-1 flex items-center justify-end px-3">
      <button
        class="w-4 h-4 p-0.5 rounded hover:bg-theme-bg-tertiary transition-colors"
        title="Download terminal text"
        on:mousedown={(event) => {
          if (event.button === 0) {
            event.stopPropagation();
            downloadTerminalText();
          }
        }}
      >
        <DownloadIcon
          class="w-full h-full text-theme-fg-secondary"
          strokeWidth={2}
        />
      </button>
    </div>
  </div>
  <div
    class="inline-block px-4 py-2 transition-opacity duration-500"
    bind:this={termEl}
    style:opacity={loaded ? 1.0 : 0.0}
    on:wheel={(event) => {
      if (focused) {
        // Don't pan the page when scrolling while the terminal is selected.
        // Conversely, we manually disable terminal scrolling unless it is currently selected.
        event.stopPropagation();
      }
    }}
  />
</div>

<style lang="postcss">
  .term-container {
    @apply inline-block rounded-lg border border-theme-border opacity-90;
    transition: transform 200ms, opacity 200ms;
  }

  .term-container:not(.focused) :global(.xterm) {
    @apply cursor-default;
  }

  .term-container.focused {
    @apply opacity-100;
  }
</style>
