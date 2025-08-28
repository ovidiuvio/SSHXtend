<script lang="ts">
  import {
    onDestroy,
    onMount,
    tick,
    beforeUpdate,
    afterUpdate,
    createEventDispatcher,
  } from "svelte";
  import { fade } from "svelte/transition";
  import { debounce, throttle } from "lodash-es";

  import { Encrypt } from "./encrypt";
  import { createLock } from "./lock";
  import { Srocket } from "./srocket";
  import type { WsClient, WsServer, WsUser, WsWinsize } from "./protocol";
  import { makeToast } from "./toast";
  import Chat, { type ChatMessage } from "./ui/Chat.svelte";
  import ChooseName from "./ui/ChooseName.svelte";
  import NameList from "./ui/NameList.svelte";
  import NetworkInfo from "./ui/NetworkInfo.svelte";
  import Settings from "./ui/Settings.svelte";
  import Toolbar from "./ui/Toolbar.svelte";
  import XTerm from "./ui/XTerm.svelte";
  import Avatars from "./ui/Avatars.svelte";
  import LiveCursor from "./ui/LiveCursor.svelte";
  import TerminalSelector from "./ui/TerminalSelector.svelte";
  import TerminalsBar from "./ui/TerminalsBar.svelte";
  import { slide } from "./action/slide";
  import { TouchZoom, INITIAL_ZOOM } from "./action/touchZoom";
  import { arrangeNewTerminal, autoArrangeTerminals } from "./arrange";
  import { settings, type ToolbarPosition, updateSettings } from "./settings";
  import { wallpaperManager } from "./wallpaper";

  export let id: string;

  const dispatch = createEventDispatcher<{ receiveName: string }>();

  // The magic numbers "left" and "top" are used to approximately center the
  // terminal at the time that it is first created.
  const CONSTANT_OFFSET_LEFT = 378;
  const CONSTANT_OFFSET_TOP = 240;

  const OFFSET_LEFT_CSS = `calc(50vw - ${CONSTANT_OFFSET_LEFT}px)`;
  const OFFSET_TOP_CSS = `calc(50vh - ${CONSTANT_OFFSET_TOP}px)`;
  const OFFSET_TRANSFORM_ORIGIN_CSS = `calc(-1 * ${OFFSET_LEFT_CSS}) calc(-1 * ${OFFSET_TOP_CSS})`;

  // Terminal width and height limits.
  const TERM_MIN_ROWS = 8;
  const TERM_MIN_COLS = 32;

  function getConstantOffset() {
    return [
      0.5 * window.innerWidth - CONSTANT_OFFSET_LEFT,
      0.5 * window.innerHeight - CONSTANT_OFFSET_TOP,
    ];
  }

  let fabricEl: HTMLElement;
  let touchZoom: TouchZoom;
  let center = [0, 0];
  let zoom = INITIAL_ZOOM;
  let initialZoomSet = false;

  let showChat = false; // @hmr:keep
  let settingsOpen = false; // @hmr:keep
  let showNetworkInfo = false; // @hmr:keep
  let showTerminalSelector = false; // @hmr:keep
  let toolbarPinned = false; // @hmr:keep
  let toolbarVisible = true;
  let toolbarHoverTimeout: number | null = null;
  
  // Terminals bar state
  let terminalsBarVisible = false; // @hmr:keep
  let terminalsBarPinned = false; // @hmr:keep
  let terminalsBarHoverTimeout: number | null = null;
  let terminalsBarUpdateInterval: number | null = null;
  
  $: toolbarPosition = $settings.toolbarPosition;
  $: terminalsBarEnabled = $settings.terminalsBarEnabled;
  $: terminalsBarPosition = $settings.terminalsBarPosition;
  
  // Manage terminals bar visibility based on enabled state
  $: {
    if (!terminalsBarEnabled) {
      terminalsBarVisible = false;
    } else if (terminalsBarPinned) {
      terminalsBarVisible = true;
      // Capture thumbnails when terminals bar becomes visible
      if (shells.length > 0) {
        captureTerminalThumbnails();
      }
    }
    // When enabled but not pinned, it should start hidden and show only on hover
  }

  // Handle live thumbnail updates when terminals bar is visible
  $: {
    if (terminalsBarEnabled && terminalsBarVisible && shells.length > 0) {
      // Start the smart update system if not already running
      if (!terminalsBarUpdateInterval) {
        startSmartThumbnailUpdates();
        // Do a full refresh when bar becomes visible + catch up on missed updates
        captureTerminalBarThumbnails();
        // Process any terminals that needed updates while bar was hidden
        for (const id of terminalNeedsUpdate) {
          updateSingleTerminalThumbnail(id);
        }
      }
    } else {
      // Stop the update interval when not visible
      if (terminalsBarUpdateInterval) {
        clearInterval(terminalsBarUpdateInterval);
        terminalsBarUpdateInterval = null;
      }
    }
  }
  
  // Wallpaper state
  $: currentWallpaper = wallpaperManager.getWallpaper($settings.wallpaperCurrent);
  $: wallpaperStyle = wallpaperManager.generateBackgroundStyle(
    $settings.wallpaperEnabled ? currentWallpaper : null,
    $settings.wallpaperFit,
    $settings.wallpaperOpacity,
    zoom,
    center
  );
  
  // Force toolbar visible when connection issues
  $: if (!connected || exitReason) {
    toolbarVisible = true;
    if (toolbarHoverTimeout) {
      clearTimeout(toolbarHoverTimeout);
      toolbarHoverTimeout = null;
    }
  }

  onMount(() => {
    touchZoom = new TouchZoom(fabricEl);
    
    // Load saved zoom level on mount
    if (!initialZoomSet && $settings.zoomLevel) {
      touchZoom.zoom = $settings.zoomLevel;
      zoom = $settings.zoomLevel;
      initialZoomSet = true;
    }
    
    touchZoom.onMove(() => {
      center = touchZoom.center;
      zoom = touchZoom.zoom;

      // Save zoom level to settings with debounce
      saveZoomLevel();

      // Blur if the user is currently focused on a terminal.
      //
      // This makes it so that panning does not stop when the cursor happens to
      // intersect with the textarea, which absorbs wheel and touch events.
      if (document.activeElement) {
        const classList = [...document.activeElement.classList];
        if (classList.includes("xterm-helper-textarea")) {
          (document.activeElement as HTMLElement).blur();
        }
      }

      showNetworkInfo = false;
    });
  });

  // Debounced save zoom level
  const saveZoomLevel = debounce(() => {
    updateSettings({ zoomLevel: zoom });
  }, 500);

  /** Returns the mouse position in infinite grid coordinates, offset transformations and zoom. */
  function normalizePosition(event: MouseEvent): [number, number] {
    const [ox, oy] = getConstantOffset();
    return [
      Math.round(center[0] + event.pageX / zoom - ox),
      Math.round(center[1] + event.pageY / zoom - oy),
    ];
  }

  let encrypt: Encrypt;
  let srocket: Srocket<WsServer, WsClient> | null = null;

  let connected = false;
  let exitReason: string | null = null;

  /** Bound "write" method for each terminal. */
  const writers: Record<number, (data: string) => void> = {};
  const termWrappers: Record<number, HTMLDivElement> = {};
  const termElements: Record<number, HTMLDivElement> = {};
  const terminalTitles: Record<number, string> = {}; // Track terminal titles
  const terminalThumbnails: Record<number, {small: string | null, large: string | null}> = {}; // Track terminal thumbnails  
  const terminalBarThumbnails: Record<number, string | null> = {}; // Track terminal bar thumbnails (320×192)
  const thumbnailGetters: Record<number, () => Promise<{small: string | null, large: string | null}>> = {}; // Terminal thumbnail getter functions
  const terminalBarThumbnailGetters: Record<number, () => Promise<string | null>> = {}; // Terminal bar thumbnail getter functions
  const terminalActivity: Record<number, number> = {}; // Track last activity timestamp for each terminal
  const terminalThumbnailHashes: Record<number, string> = {}; // Track content hashes to detect changes
  const terminalUpdateTimeouts: Record<number, number> = {}; // Track pending thumbnail updates
  const terminalNeedsUpdate: Set<number> = new Set(); // Track which terminals need updates
  let sharedThumbnailCanvas: HTMLCanvasElement | null = null; // Reuse canvas for performance
  let activeThumbnailGenerations = 0; // Track concurrent thumbnail operations
  const MAX_CONCURRENT_THUMBNAILS = 1; // Strict limit to prevent main thread blocking
  let isUIBusy = false; // Track if terminals are being moved/resized
  const chunknums: Record<number, number> = {};
  const locks: Record<number, any> = {};
  let userId = 0;
  let users: [number, WsUser][] = [];
  let shells: [number, WsWinsize][] = [];
  let subscriptions = new Set<number>();

  // May be undefined before `users` is first populated.
  $: hasWriteAccess = users.find(([uid]) => uid === userId)?.[1]?.canWrite;

  let moving = -1; // Terminal ID that is being dragged.
  let movingOrigin = [0, 0]; // Coordinates of mouse at origin when drag started.
  let movingSize: WsWinsize; // New [x, y] position of the dragged terminal.
  let movingIsDone = false; // Moving finished but hasn't been acknowledged.

  let resizing = -1; // Terminal ID that is being resized.
  let resizingOrigin = [0, 0]; // Coordinates of top-left origin when resize started.
  let resizingCell = [0, 0]; // Pixel dimensions of a single terminal cell.
  let resizingSize: WsWinsize; // Last resize message sent.

  // Track UI busy state to prevent thumbnail generation during drag operations
  $: isUIBusy = moving !== -1 || resizing !== -1;

  let chatMessages: ChatMessage[] = [];
  let newMessages = false;

  let serverLatencies: number[] = [];
  let shellLatencies: number[] = [];

  onMount(async () => {
    // The page hash sets the end-to-end encryption key.
    const key = window.location.hash?.slice(1).split(",")[0] ?? "";
    const writePassword = window.location.hash?.slice(1).split(",")[1] ?? null;

    encrypt = await Encrypt.new(key);
    const encryptedZeros = await encrypt.zeros();

    const writeEncryptedZeros = writePassword
      ? await (await Encrypt.new(writePassword)).zeros()
      : null;

    srocket = new Srocket<WsServer, WsClient>(`/api/s/${id}`, {
      onMessage(message) {
        if (message.hello) {
          userId = message.hello[0];
          dispatch("receiveName", message.hello[1]);
          makeToast({
            kind: "success",
            message: `Connected to the server.`,
          });
          exitReason = null;
        } else if (message.invalidAuth) {
          exitReason =
            "The URL is not correct, invalid end-to-end encryption key.";
          srocket?.dispose();
        } else if (message.chunks) {
          let [id, seqnum, chunks] = message.chunks;
          locks[id](async () => {
            await tick();
            chunknums[id] += chunks.length;
            for (const data of chunks) {
              const buf = await encrypt.segment(
                0x100000000n | BigInt(id),
                BigInt(seqnum),
                data,
              );
              seqnum += data.length;
              writers[id](new TextDecoder().decode(buf));
              // Track terminal activity and trigger immediate thumbnail update
              terminalActivity[id] = Date.now();
              // Schedule throttled thumbnail update for terminals bar
              if (terminalsBarEnabled && terminalsBarVisible) {
                scheduleThrottledThumbnailUpdate(id);
              }
            }
          });
        } else if (message.users) {
          users = message.users;
        } else if (message.userDiff) {
          const [id, update] = message.userDiff;
          users = users.filter(([uid]) => uid !== id);
          if (update !== null) {
            users = [...users, [id, update]];
          }
        } else if (message.shells) {
          shells = message.shells;
          if (movingIsDone) {
            moving = -1;
          }
          for (const [id] of message.shells) {
            if (!subscriptions.has(id)) {
              chunknums[id] ??= 0;
              locks[id] ??= createLock();
              subscriptions.add(id);
              srocket?.send({ subscribe: [id, chunknums[id]] });
            }
          }
        } else if (message.hear) {
          const [uid, name, msg] = message.hear;
          chatMessages.push({ uid, name, msg, sentAt: new Date() });
          chatMessages = chatMessages;
          if (!showChat) newMessages = true;
        } else if (message.shellLatency !== undefined) {
          const shellLatency = Number(message.shellLatency);
          shellLatencies = [...shellLatencies, shellLatency].slice(-10);
        } else if (message.pong !== undefined) {
          const serverLatency = Date.now() - Number(message.pong);
          serverLatencies = [...serverLatencies, serverLatency].slice(-10);
        } else if (message.error) {
          console.warn("Server error: " + message.error);
        }
      },

      onConnect() {
        srocket?.send({ authenticate: [encryptedZeros, writeEncryptedZeros] });
        if ($settings.name) {
          srocket?.send({ setName: $settings.name });
        }
        connected = true;
      },

      onDisconnect() {
        connected = false;
        subscriptions.clear();
        users = [];
        serverLatencies = [];
        shellLatencies = [];
      },

      onClose(event) {
        if (event.code === 4404) {
          exitReason = "Failed to connect: " + event.reason;
        } else if (event.code === 4500) {
          exitReason = "Internal server error: " + event.reason;
        }
      },
    });
  });

  onDestroy(() => srocket?.dispose());

  // Send periodic ping messages for latency estimation.
  onMount(() => {
    const pingIntervalId = window.setInterval(() => {
      if (srocket?.connected) {
        srocket.send({ ping: BigInt(Date.now()) });
      }
    }, 2000);
    return () => window.clearInterval(pingIntervalId);
  });

  function integerMedian(values: number[]) {
    if (values.length === 0) {
      return null;
    }
    const sorted = values.toSorted();
    const mid = Math.floor(sorted.length / 2);
    return sorted.length % 2 !== 0
      ? sorted[mid]
      : Math.round((sorted[mid - 1] + sorted[mid]) / 2);
  }

  $: if ($settings.name) {
    srocket?.send({ setName: $settings.name });
  }

  let counter = 0n;

  async function handleCreate() {
    if (hasWriteAccess === false) {
      makeToast({
        kind: "info",
        message: "You are in read-only mode and cannot create new terminals.",
      });
      return;
    }
    if (shells.length >= 14) {
      makeToast({
        kind: "error",
        message: "You can only create up to 14 terminals.",
      });
      return;
    }
    const existing = shells.map(([id, winsize]) => ({
      x: winsize.x,
      y: winsize.y,
      width: termWrappers[id].clientWidth,
      height: termWrappers[id].clientHeight,
    }));
    const { x, y } = arrangeNewTerminal(existing);
    srocket?.send({ create: [x, y] });
    touchZoom.moveTo([x, y], INITIAL_ZOOM);
  }
  

  async function handleInput(id: number, data: Uint8Array) {
    if (counter === 0n) {
      // On the first call, initialize the counter to a random 64-bit integer.
      const array = new Uint8Array(8);
      crypto.getRandomValues(array);
      counter = new DataView(array.buffer).getBigUint64(0);
    }
    const offset = counter;
    counter += BigInt(data.length); // Must increment before the `await`.
    const encrypted = await encrypt.segment(0x200000000n, offset, data);
    srocket?.send({ data: [id, encrypted, offset] });
    
    // Track terminal activity and trigger immediate thumbnail update
    terminalActivity[id] = Date.now();
    // Schedule throttled thumbnail update for terminals bar  
    if (terminalsBarEnabled && terminalsBarVisible) {
      scheduleThrottledThumbnailUpdate(id);
    }
  }

  // Stupid hack to preserve input focus when terminals are reordered.
  // See: https://github.com/sveltejs/svelte/issues/3973
  let activeElement: Element | null = null;

  beforeUpdate(() => {
    activeElement = document.activeElement;
  });

  afterUpdate(() => {
    if (activeElement instanceof HTMLElement) activeElement.focus();
  });

  // Global mouse handler logic follows, attached to the window element for smoothness.
  onMount(() => {
    // 50 milliseconds between successive terminal move updates.
    const sendMove = throttle((message: WsClient) => {
      srocket?.send(message);
    }, 50);

    // 80 milliseconds between successive cursor updates.
    const sendCursor = throttle((message: WsClient) => {
      srocket?.send(message);
    }, 80);

    function handleMouse(event: MouseEvent) {
      if (moving !== -1 && !movingIsDone) {
        const [x, y] = normalizePosition(event);
        movingSize = {
          ...movingSize,
          x: Math.round(x - movingOrigin[0]),
          y: Math.round(y - movingOrigin[1]),
        };
        sendMove({ move: [moving, movingSize] });
      }

      if (resizing !== -1) {
        const cols = Math.max(
          Math.floor((event.pageX - resizingOrigin[0]) / resizingCell[0]),
          TERM_MIN_COLS, // Minimum number of columns.
        );
        const rows = Math.max(
          Math.floor((event.pageY - resizingOrigin[1]) / resizingCell[1]),
          TERM_MIN_ROWS, // Minimum number of rows.
        );
        if (rows !== resizingSize.rows || cols !== resizingSize.cols) {
          resizingSize = { ...resizingSize, rows, cols };
          srocket?.send({ move: [resizing, resizingSize] });
        }
      }

      sendCursor({ setCursor: normalizePosition(event) });
    }

    function handleMouseEnd(event: MouseEvent) {
      if (moving !== -1) {
        movingIsDone = true;
        sendMove.cancel();
        srocket?.send({ move: [moving, movingSize] });
      }
      

      if (resizing !== -1) {
        resizing = -1;
      }

      if (event.type === "mouseleave") {
        sendCursor.cancel();
        srocket?.send({ setCursor: null });
      }
    }

    window.addEventListener("mousemove", handleMouse);
    window.addEventListener("mouseup", handleMouseEnd);
    document.body.addEventListener("mouseleave", handleMouseEnd);
    return () => {
      window.removeEventListener("mousemove", handleMouse);
      window.removeEventListener("mouseup", handleMouseEnd);
      document.body.removeEventListener("mouseleave", handleMouseEnd);
    };
  });

  let focused: number[] = [];
  $: setFocus(focused);

  // Wait a small amount of time, since blur events happen before focus events.
  const setFocus = debounce((focused: number[]) => {
    srocket?.send({ setFocus: focused[0] ?? null });
  }, 20);

  function handleToolbarMouseEnter() {
    if (toolbarHoverTimeout) {
      clearTimeout(toolbarHoverTimeout);
      toolbarHoverTimeout = null;
    }
    toolbarVisible = true;
  }

  function handleToolbarMouseLeave() {
    if (!toolbarPinned) {
      toolbarHoverTimeout = window.setTimeout(() => {
        toolbarVisible = false;
        toolbarHoverTimeout = null;
      }, 500);
    }
  }

  function handleTogglePin() {
    toolbarPinned = !toolbarPinned;
    if (toolbarPinned) {
      toolbarVisible = true;
      if (toolbarHoverTimeout) {
        clearTimeout(toolbarHoverTimeout);
        toolbarHoverTimeout = null;
      }
    }
  }

  function handleZoomIn() {
    const newZoom = Math.min(zoom * 1.2, 4);
    touchZoom.zoom = newZoom;
    zoom = newZoom;
    updateSettings({ zoomLevel: newZoom });
  }

  function handleZoomOut() {
    const newZoom = Math.max(zoom / 1.2, 0.25);
    touchZoom.zoom = newZoom;
    zoom = newZoom;
    updateSettings({ zoomLevel: newZoom });
  }

  function handleZoomReset() {
    touchZoom.zoom = 1;
    zoom = 1;
    updateSettings({ zoomLevel: 1 });
  }

  function handleAutoArrange() {
    if (shells.length === 0) {
      return;
    }

    // Prepare terminal data for auto-arrange with actual dimensions
    const terminals = shells.map(([id, winsize]) => {
      const wrapper = termWrappers[id];
      return {
        id,
        x: winsize.x,
        y: winsize.y,
        rows: winsize.rows,
        cols: winsize.cols,
        width: wrapper ? wrapper.clientWidth : undefined,
        height: wrapper ? wrapper.clientHeight : undefined,
      };
    });

    // Calculate new positions
    const newPositions = autoArrangeTerminals(terminals);

    // Apply new positions to each terminal
    newPositions.forEach((position, id) => {
      const shell = shells.find(([shellId]) => shellId === id);
      if (shell) {
        const [shellId, winsize] = shell;
        const newWinsize = { ...winsize, x: position.x, y: position.y };
        srocket?.send({ move: [shellId, newWinsize] });
      }
    });

    // Center view on the arranged terminals if positions changed
    if (newPositions.size > 0) {
      const positions = Array.from(newPositions.values());
      const avgX = positions.reduce((sum, p) => sum + p.x, 0) / positions.length;
      const avgY = positions.reduce((sum, p) => sum + p.y, 0) / positions.length;
      
      // Get average terminal dimensions for centering
      const avgWidth = terminals.reduce((sum, t) => sum + (t.width || 752), 0) / terminals.length;
      const avgHeight = terminals.reduce((sum, t) => sum + (t.height || 515), 0) / terminals.length;
      
      touchZoom.moveTo([avgX + avgWidth / 2, avgY + avgHeight / 2], 1);
    }
  }

  function handleCenterTerminal(id: number, winsize?: WsWinsize) {
    const shell = shells.find(([shellId]) => shellId === id);
    if (!shell) return;

    const [shellId, currentWinsize] = shell;
    const ws = winsize || currentWinsize;
    const wrapper = termWrappers[id];

    // Get actual terminal dimensions
    const width = wrapper ? wrapper.clientWidth : 752;
    const height = wrapper ? wrapper.clientHeight : 515;

    // Center on the terminal's visual center
    const terminalCenterX = ws.x + width / 2 - CONSTANT_OFFSET_LEFT;
    const terminalCenterY = ws.y + height / 2 - CONSTANT_OFFSET_TOP;

    touchZoom.moveTo([terminalCenterX, terminalCenterY], INITIAL_ZOOM);

    // Focus the terminal after centering
    setTimeout(() => {
      const termElement = termElements[id];
      if (termElement) {
        const textarea =
          termElement.querySelector(".xterm-helper-textarea") as HTMLTextAreaElement;
        if (textarea) {
          textarea.focus();
        }
      }
    }, 300);
  }

  function handleTerminalSelectorSelect(event: CustomEvent<{ id: number, winsize: WsWinsize }>) {
    handleCenterTerminal(event.detail.id, event.detail.winsize);
    showTerminalSelector = false;
  }

  // Terminals bar event handlers
  function handleTerminalsBarSelect(event: CustomEvent<{ id: number, winsize: WsWinsize }>) {
    handleCenterTerminal(event.detail.id, event.detail.winsize);
  }

  function handleTerminalsBarTogglePin() {
    terminalsBarPinned = !terminalsBarPinned;
    if (terminalsBarPinned) {
      terminalsBarVisible = true;
      if (terminalsBarHoverTimeout) {
        clearTimeout(terminalsBarHoverTimeout);
        terminalsBarHoverTimeout = null;
      }
      // Capture thumbnails immediately when pinning
      if (shells.length > 0) {
        captureTerminalThumbnails();
      }
    }
  }

  function handleTerminalsBarMouseEnter() {
    if (terminalsBarHoverTimeout) {
      clearTimeout(terminalsBarHoverTimeout);
      terminalsBarHoverTimeout = null;
    }
    terminalsBarVisible = true;
  }

  function handleTerminalsBarMouseLeave() {
    if (!terminalsBarPinned) {
      terminalsBarHoverTimeout = window.setTimeout(() => {
        terminalsBarVisible = false;
        terminalsBarHoverTimeout = null;
      }, 500);
    }
  }

  function handleTerminalsBarClose() {
    if (!terminalsBarPinned) {
      terminalsBarVisible = false;
    }
  }

  // Global keyboard shortcuts
  async function handleGlobalKeydown(event: KeyboardEvent) {
    const isMac = navigator.platform.startsWith('Mac');
    
    // Use Ctrl+` (or Cmd+` on Mac) for terminal selector
    if (event.key === '`' && 
        ((isMac && event.metaKey) || (!isMac && event.ctrlKey)) && 
        !event.shiftKey && !event.altKey) {
      event.preventDefault();
      event.stopPropagation();
      
      // Only open if we have terminals
      if (shells.length > 0) {
        // Capture thumbnails before showing selector
        await captureTerminalThumbnails();
        showTerminalSelector = true;
      }
    }
  }

  async function handleResize() {
    if (showTerminalSelector || (terminalsBarEnabled && terminalsBarVisible)) {
      await captureTerminalThumbnails();
    }
  }

  onMount(() => {
    window.addEventListener('keydown', handleGlobalKeydown);
    window.addEventListener('resize', handleResize);
    return () => {
      window.removeEventListener('keydown', handleGlobalKeydown);
      window.removeEventListener('resize', handleResize);
    };
  });

  // Cleanup intervals on component destruction
  onDestroy(() => {
    if (terminalsBarUpdateInterval) {
      clearInterval(terminalsBarUpdateInterval);
      terminalsBarUpdateInterval = null;
    }
  });
  // Track terminal title changes
  function handleTerminalTitleChange(id: number, title: string) {
    terminalTitles[id] = title;
  }
  
  // Capture terminal thumbnails when opening selector
  async function captureTerminalThumbnails() {
    for (const [id] of shells) {
      const getter = thumbnailGetters[id];
      if (getter) {
        try {
          terminalThumbnails[id] = await getter();
        } catch (error) {
          console.warn(`Failed to capture thumbnail for terminal ${id}:`, error);
          terminalThumbnails[id] = {small: null, large: null};
        }
      }
    }
  }

  // Capture bar thumbnails for all terminals (for terminals bar)
  async function captureTerminalBarThumbnails() {
    for (const [id] of shells) {
      await updateSingleTerminalThumbnail(id);
    }
  }

  // Update thumbnail for a single terminal with concurrency control and non-blocking processing
  async function updateSingleTerminalThumbnail(id: number, retryCount = 0) {
    // Critical performance checks - skip if would block main thread
    if (!terminalsBarEnabled || !terminalsBarVisible || isUIBusy) {
      terminalNeedsUpdate.add(id);
      return;
    }

    // Strict concurrency limit to prevent main thread blocking
    if (activeThumbnailGenerations >= MAX_CONCURRENT_THUMBNAILS) {
      terminalNeedsUpdate.add(id);
      // Schedule retry after current operations complete
      setTimeout(() => updateSingleTerminalThumbnail(id, retryCount), 50);
      return;
    }

    const getter = terminalBarThumbnailGetters[id];
    if (!getter) return;
    
    activeThumbnailGenerations++;
    
    try {
      // Yield to allow other operations (critical for network/pings)
      await new Promise(resolve => setTimeout(resolve, 0));
      
      const thumbnail = await getter();
      
      // Yield again after potentially heavy canvas operation
      await new Promise(resolve => setTimeout(resolve, 0));
      
      if (thumbnail) {
        terminalBarThumbnails[id] = thumbnail;
        terminalNeedsUpdate.delete(id); // Successfully updated
      } else if (retryCount < 2) {
        // Retry failed capture after short delay
        setTimeout(() => updateSingleTerminalThumbnail(id, retryCount + 1), 100);
      }
    } catch (error) {
      if (retryCount < 2) {
        // Retry on error after short delay  
        setTimeout(() => updateSingleTerminalThumbnail(id, retryCount + 1), 100);
      } else {
        console.warn(`Failed to capture bar thumbnail for terminal ${id} after retries:`, error);
        terminalBarThumbnails[id] = null;
      }
    } finally {
      activeThumbnailGenerations--;
    }
  }

  // Throttled thumbnail update with recovery for missed updates
  function scheduleThrottledThumbnailUpdate(id: number) {
    // Mark terminal as needing update
    terminalNeedsUpdate.add(id);
    
    // Clear any existing timeout for this terminal
    if (terminalUpdateTimeouts[id]) {
      clearTimeout(terminalUpdateTimeouts[id]);
    }
    
    // Schedule update with 1 second delay (throttle high-frequency changes)
    terminalUpdateTimeouts[id] = window.setTimeout(() => {
      delete terminalUpdateTimeouts[id];
      updateSingleTerminalThumbnail(id); // Always attempt update, handle visibility inside
    }, 1000);
  }

  // No periodic updates needed - thumbnails update immediately on data changes
  function startSmartThumbnailUpdates() {
    // All updates happen immediately when terminal data changes
    // No periodic timer needed
    terminalsBarUpdateInterval = null;
  }
</script>

<!-- Wheel handler stops native macOS Chrome zooming on pinch. -->
<main
  class="p-8"
  class:cursor-nwse-resize={resizing !== -1}
  on:wheel={(event) => event.preventDefault()}
>
  <div
    class="absolute z-10 transition-all duration-300 ease-in-out flex"
    class:inset-x-0={toolbarPosition === "top" || toolbarPosition === "bottom"}
    class:inset-y-0={toolbarPosition === "left" || toolbarPosition === "right"}
    class:justify-center={toolbarPosition === "top" || toolbarPosition === "bottom"}
    class:items-center={toolbarPosition === "left" || toolbarPosition === "right"}
    class:top-8={toolbarPosition === "top" && toolbarVisible}
    class:top-0={toolbarPosition === "top" && !toolbarVisible}
    class:bottom-8={toolbarPosition === "bottom" && toolbarVisible}
    class:bottom-0={toolbarPosition === "bottom" && !toolbarVisible}
    class:left-8={toolbarPosition === "left" && toolbarVisible}
    class:left-0={toolbarPosition === "left" && !toolbarVisible}
    class:right-8={toolbarPosition === "right" && toolbarVisible}
    class:right-0={toolbarPosition === "right" && !toolbarVisible}
    class:opacity-100={toolbarVisible}
    class:opacity-0={!toolbarVisible}
    class:pointer-events-none={!toolbarVisible}
    on:mouseenter={handleToolbarMouseEnter}
    on:mouseleave={handleToolbarMouseLeave}
  >
    <div class="pointer-events-auto">
      <Toolbar
        {connected}
        {exitReason}
        {newMessages}
        {hasWriteAccess}
        pinned={toolbarPinned}
        position={toolbarPosition}
        zoomLevel={zoom}
        on:create={handleCreate}
        on:chat={() => {
          showChat = !showChat;
          newMessages = false;
        }}
        on:settings={() => {
          settingsOpen = true;
        }}
        on:networkInfo={() => {
          showNetworkInfo = !showNetworkInfo;
        }}
        on:togglePin={handleTogglePin}
        on:zoomIn={handleZoomIn}
        on:zoomOut={handleZoomOut}
        on:zoomReset={handleZoomReset}
        on:autoArrange={handleAutoArrange}
        on:terminalSelector={async () => {
          await captureTerminalThumbnails();
          showTerminalSelector = true;
        }}
      />
    </div>

    {#if showNetworkInfo}
      <div 
        class="absolute pointer-events-auto"
        class:top-20={toolbarPosition === "top"}
        class:bottom-20={toolbarPosition === "bottom"}
        class:left-20={toolbarPosition === "left"}
        class:right-20={toolbarPosition === "right"}
        class:translate-x-[116.5px]={toolbarPosition === "top" || toolbarPosition === "bottom"}
        class:translate-y-[116.5px]={toolbarPosition === "left" || toolbarPosition === "right"}
      >
        <NetworkInfo
          status={connected
            ? "connected"
            : exitReason
            ? "no-shell"
            : "no-server"}
          serverLatency={integerMedian(serverLatencies)}
          shellLatency={integerMedian(shellLatencies)}
        />
      </div>
    {/if}
  </div>

  <!-- Invisible hover zones for showing the toolbar based on position -->
  {#if !toolbarPinned && !toolbarVisible}
    {#if toolbarPosition === "top"}
      <div
        class="absolute top-0 inset-x-0 h-8 z-10"
        on:mouseenter={handleToolbarMouseEnter}
      />
    {:else if toolbarPosition === "bottom"}
      <div
        class="absolute bottom-0 inset-x-0 h-8 z-10"
        on:mouseenter={handleToolbarMouseEnter}
      />
    {:else if toolbarPosition === "left"}
      <div
        class="absolute left-0 inset-y-0 w-8 z-10"
        on:mouseenter={handleToolbarMouseEnter}
      />
    {:else if toolbarPosition === "right"}
      <div
        class="absolute right-0 inset-y-0 w-8 z-10"
        on:mouseenter={handleToolbarMouseEnter}
      />
    {/if}
  {/if}

  <!-- Invisible hover zones for showing the terminals bar based on position -->
  {#if terminalsBarEnabled && !terminalsBarPinned && !terminalsBarVisible}
    {#if terminalsBarPosition === "top" && terminalsBarPosition !== toolbarPosition}
      <div
        class="absolute top-0 inset-x-0 h-8 z-10"
        on:mouseenter={handleTerminalsBarMouseEnter}
      />
    {:else if terminalsBarPosition === "bottom" && terminalsBarPosition !== toolbarPosition}
      <div
        class="absolute bottom-0 inset-x-0 h-8 z-10"
        on:mouseenter={handleTerminalsBarMouseEnter}
      />
    {:else if terminalsBarPosition === "left" && terminalsBarPosition !== toolbarPosition}
      <div
        class="absolute left-0 inset-y-0 w-8 z-10"
        on:mouseenter={handleTerminalsBarMouseEnter}
      />
    {:else if terminalsBarPosition === "right" && terminalsBarPosition !== toolbarPosition}
      <div
        class="absolute right-0 inset-y-0 w-8 z-10"
        on:mouseenter={handleTerminalsBarMouseEnter}
      />
    {:else if terminalsBarPosition === toolbarPosition}
      <!-- When terminals bar is at same position as toolbar, create offset hover zone -->
      {#if terminalsBarPosition === "top"}
        <div
          class="absolute inset-x-0 h-8 z-10"
          style="top: 6rem;"
          on:mouseenter={handleTerminalsBarMouseEnter}
        />
      {:else if terminalsBarPosition === "bottom"}
        <div
          class="absolute inset-x-0 h-8 z-10"
          style="bottom: 6rem;"
          on:mouseenter={handleTerminalsBarMouseEnter}
        />
      {:else if terminalsBarPosition === "left"}
        <div
          class="absolute inset-y-0 w-8 z-10"
          style="left: 4rem;"
          on:mouseenter={handleTerminalsBarMouseEnter}
        />
      {:else if terminalsBarPosition === "right"}
        <div
          class="absolute inset-y-0 w-8 z-10"
          style="right: 4rem;"
          on:mouseenter={handleTerminalsBarMouseEnter}
        />
      {/if}
    {/if}
  {/if}

  {#if showChat}
    <div
      class="absolute flex flex-col justify-end inset-y-4 right-4 w-80 pointer-events-none z-10"
    >
      <Chat
        {userId}
        messages={chatMessages}
        on:chat={(event) => srocket?.send({ chat: event.detail })}
        on:close={() => (showChat = false)}
      />
    </div>
  {/if}

  <Settings open={settingsOpen} on:close={() => (settingsOpen = false)} />

  <ChooseName />

  <!--
    Wallpaper or dotted pattern background appears underneath the rest of the elements,
    moves and zooms with the fabric of the canvas.
  -->
  <div
    class="absolute inset-0 -z-10"
    style="{wallpaperStyle}"
  />

  <!-- User list -->
  <div class="fixed top-4 left-4 z-10">
    <NameList {users} />
  </div>

  <div class="absolute inset-0 overflow-hidden touch-none" bind:this={fabricEl}>
    {#each shells as [id, winsize] (id)}
      {@const ws = id === moving ? movingSize : winsize}
      <div
        class="absolute"
        style:left={OFFSET_LEFT_CSS}
        style:top={OFFSET_TOP_CSS}
        style:transform-origin={OFFSET_TRANSFORM_ORIGIN_CSS}
        transition:fade|local
        use:slide={{ x: ws.x, y: ws.y, center, zoom, immediate: id === moving }}
        bind:this={termWrappers[id]}
      >
        <XTerm
          rows={ws.rows}
          cols={ws.cols}
          bind:write={writers[id]}
          bind:getThumbnails={thumbnailGetters[id]}
          bind:getThumbnailForBar={terminalBarThumbnailGetters[id]}
          bind:termEl={termElements[id]}
          on:data={({ detail: data }) =>
            hasWriteAccess && handleInput(id, data)}
          on:close={() => srocket?.send({ close: id })}
          on:shrink={() => {
            if (!hasWriteAccess) return;
            const rows = Math.max(ws.rows - 4, TERM_MIN_ROWS);
            const cols = Math.max(ws.cols - 10, TERM_MIN_COLS);
            if (rows !== ws.rows || cols !== ws.cols) {
              srocket?.send({ move: [id, { ...ws, rows, cols }] });
            }
          }}
          on:expand={() => {
            if (!hasWriteAccess) return;
            const rows = ws.rows + 4;
            const cols = ws.cols + 10;
            srocket?.send({ move: [id, { ...ws, rows, cols }] });
          }}
          on:bringToFront={() => {
            if (!hasWriteAccess) return;
            showNetworkInfo = false;
            srocket?.send({ move: [id, null] });
          }}
          on:startMove={({ detail: event }) => {
            if (!hasWriteAccess) return;
            const [x, y] = normalizePosition(event);
            moving = id;
            movingOrigin = [x - ws.x, y - ws.y];
            movingSize = ws;
            movingIsDone = false;
          }}
          on:focus={() => {
            if (!hasWriteAccess) return;
            focused = [...focused, id];
          }}
          on:blur={() => {
            focused = focused.filter((i) => i !== id);
          }}
          on:titleChange={({ detail: title }) => handleTerminalTitleChange(id, title)}
        />

        <!-- User avatars -->
        <div class="absolute bottom-2.5 right-2.5 pointer-events-none">
          <Avatars
            users={users.filter(
              ([uid, user]) => uid !== userId && user.focus === id,
            )}
          />
        </div>

        <!-- Interactable element for resizing -->
        <div
          class="absolute w-5 h-5 -bottom-1 -right-1 cursor-nwse-resize"
          on:mousedown={(event) => {
            const canvasEl = termElements[id].querySelector(".xterm-screen");
            if (canvasEl) {
              resizing = id;
              const r = canvasEl.getBoundingClientRect();
              resizingOrigin = [event.pageX - r.width, event.pageY - r.height];
              resizingCell = [r.width / ws.cols, r.height / ws.rows];
              resizingSize = ws;
            }
          }}
          on:pointerdown={(event) => event.stopPropagation()}
        />
      </div>
    {/each}
    

    {#each users.filter(([id, user]) => id !== userId && user.cursor !== null) as [id, user] (id)}
      <div
        class="absolute"
        style:left={OFFSET_LEFT_CSS}
        style:top={OFFSET_TOP_CSS}
        style:transform-origin={OFFSET_TRANSFORM_ORIGIN_CSS}
        transition:fade|local={{ duration: 200 }}
        use:slide={{
          x: user.cursor?.[0] ?? 0,
          y: user.cursor?.[1] ?? 0,
          center,
          zoom,
        }}
      >
        <LiveCursor {user} />
      </div>
    {/each}
  </div>
  
  <!-- Terminal Selector Overlay -->
  {#if showTerminalSelector}
    <TerminalSelector
      {shells}
      focusedTerminals={focused}
      {terminalTitles}
      terminalThumbnails={terminalThumbnails}
      on:select={handleTerminalSelectorSelect}
      on:close={() => showTerminalSelector = false}
    />
  {/if}

  <!-- Terminals Bar -->
  {#if terminalsBarEnabled && terminalsBarVisible}
    <TerminalsBar
      {shells}
      focusedTerminals={focused}
      {terminalTitles}
      terminalThumbnails={terminalBarThumbnails}
      position={terminalsBarPosition}
      mainToolbarPosition={toolbarPosition}
      pinned={terminalsBarPinned}
      visible={terminalsBarVisible}
      on:select={handleTerminalsBarSelect}
      on:togglePin={handleTerminalsBarTogglePin}
      on:mouseenter={handleTerminalsBarMouseEnter}
      on:mouseleave={handleTerminalsBarMouseLeave}
    />
  {/if}
</main>
