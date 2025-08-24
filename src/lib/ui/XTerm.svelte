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
  import { DownloadIcon, CodeIcon, FileTextIcon } from "svelte-feather-icons";
  import SparklesIcon from "./icons/SparklesIcon.svelte";
  import SparklesNewIcon from "./icons/SparklesNewIcon.svelte";

  import themes from "./themes";
  import CircleButton from "./CircleButton.svelte";
  import CircleButtons from "./CircleButtons.svelte";
  import { settings, type CopyFormat } from "$lib/settings";
  import { TypeAheadAddon } from "$lib/typeahead";
  import { geminiService } from "$lib/gemini";
  import { openRouterService } from "$lib/openrouter";
  import { markdownToAnsi } from "$lib/markdownToAnsi";
  import { contextManager } from "$lib/contextManager";
  import { ExportManager, type ExportFormat, type ExportOptions } from "$lib/export";
  import ExportModal from "./ExportModal.svelte";

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
    titleChange: string;
  }>();

  const typeahead = new TypeAheadAddon();

  export let rows: number, cols: number;
  export let write: (data: string) => void; // bound function prop
  export let getThumbnail: () => Promise<string | null> = async () => null; // bound function prop
  export let getThumbnails: () => Promise<{small: string | null, large: string | null}> = async () => ({small: null, large: null}); // bound function prop

  export let termEl: HTMLDivElement = null as any; // suppress "missing prop" warning
  let term: Terminal | null = null;

  $: theme = themes[$settings.theme];

  $: if (term) {
    // If the theme changes, update existing terminals' appearance.
    term.options.theme = theme;
    term.options.scrollback = $settings.scrollback;
    term.options.fontFamily = $settings.fontFamily;
    term.options.fontSize = $settings.fontSize;
    term.options.fontWeight = $settings.fontWeight;
    term.options.fontWeightBold = $settings.fontWeightBold;
    
    // Update export manager when settings change
    updateExportManager();
  }

  async function copyToClipboard(text: string) {
    if (!text) return;
    
    try {
      await navigator.clipboard.writeText(text);
    } catch (err) {
      console.error('Failed to copy text to clipboard:', err);
    }
  }

  async function copyContentToClipboard(format: CopyFormat = $settings.copyButtonFormat) {
    if (!exportManager) {
      console.error('Export manager not initialized');
      return;
    }

    try {
      // Export terminal content in the specified format
      const result = await exportManager.export({ format, selectionOnly: false });
      
      if (result && typeof result.content === 'string') {
        await navigator.clipboard.writeText(result.content);
        const formatNames = {
          html: 'HTML',
          ansi: 'ANSI',
          txt: 'text',
          markdown: 'Markdown'
        };
        makeToast({
          kind: "success",
          message: `Terminal ${formatNames[format]} copied to clipboard`,
        });
      }
    } catch (error) {
      console.error(`Failed to copy ${format} to clipboard:`, error);
      makeToast({
        kind: "error",
        message: `Failed to copy ${format}: ${error instanceof Error ? error.message : 'Unknown error'}`,
      });
    }
  }

  let loaded = false;
  let focused = false;
  let currentTitle = "Remote Terminal";
  let exportManager: ExportManager | null = null;
  let showExportModal = false;
  
  // Always use FileText icon regardless of format
  $: copyButtonIcon = FileTextIcon;
  
  $: copyButtonTooltip = (() => {
    const formats = {
      html: 'Copy terminal as HTML to clipboard',
      ansi: 'Copy terminal as ANSI to clipboard',
      txt: 'Copy terminal as plain text to clipboard',
      markdown: 'Copy terminal as Markdown to clipboard'
    };
    return formats[$settings.copyButtonFormat];
  })();
  
  // AI state for this terminal instance
  interface AIState {
    isProcessingAI: boolean;
    conversationHistory: Array<{role: string, content: string}>;
    selectedTextForAI: string;
    aiCommandBuffer: string;
    isInAIMode: boolean;
  }
  
  let aiState: AIState = {
    isProcessingAI: false,
    conversationHistory: [],
    selectedTextForAI: "",
    aiCommandBuffer: "",
    isInAIMode: false
  };
  
  function formatMarkdownForTerminal(markdown: string, terminal: Terminal) {
    try {
      // Use our lightweight markdown to ANSI converter
      const ansiFormatted = markdownToAnsi(markdown);
      
      // Split by lines and write to terminal with proper carriage returns
      const lines = ansiFormatted.split('\n');
      for (const line of lines) {
        terminal.write(line + '\r\n');
      }
    } catch (error) {
      // Fallback to raw output if parsing fails
      console.error('Markdown parsing error:', error);
      const lines = markdown.split('\n');
      for (const line of lines) {
        terminal.write(line + '\r\n');
      }
    }
  }
  
  async function processAIQuery(query: string, initialContext: string = "", showQuery: boolean = true) {
    if (!term || aiState.isProcessingAI) return;
    
    aiState.isProcessingAI = true;
    
    // Track if this is a new conversation (before we potentially add to history)
    const isNewConversation = aiState.conversationHistory.length === 0;
    
    // Show the user's query as "committed" if not already shown
    if (showQuery && query) {
      term.write('\x1b[32m> \x1b[0m' + query + '\r\n');
    }
    
    // Show loading indicator with icon on new line
    term.write('\x1b[36m🔄  Thinking...\x1b[0m\r\n');
    
    try {
      // Check if we need to compress the conversation before adding more
      const contextStatus = contextManager.getContextStatus(aiState.conversationHistory);
      
      if (contextStatus.shouldCompress && $settings.aiAutoCompress) {
        // Show compression notification
        term.write('\x1b[33m📦 Compressing conversation history to fit context window...\x1b[0m\r\n');
        
        // Compress the conversation
        aiState.conversationHistory = await contextManager.checkAndCompress(aiState.conversationHistory);
        
        // Clear and redraw the compression message
        term.write('\x1b[1A\x1b[2K'); // Move up and clear line
        term.write('\x1b[32m✅ Conversation compressed\x1b[0m\r\n');
        
        // Small delay for user to see the message
        await new Promise(resolve => setTimeout(resolve, 500));
        term.write('\x1b[1A\x1b[2K'); // Clear the success message too
      }
      // Build the full context including conversation history
      let fullContext = "";
      let needsInitialContext = false;
      
      // Check if we need initial context (first time or empty history)
      if (aiState.conversationHistory.length === 0) {
        needsInitialContext = true;
        
        if (initialContext) {
          // Use provided initial context
          fullContext = `Terminal context:\n${initialContext}\n\n`;
        } else {
          // Get terminal buffer as context
          const buffer = term.buffer.active;
          const lines = [];
          const startLine = Math.max(0, buffer.cursorY - 50);
          for (let i = startLine; i <= buffer.cursorY; i++) {
            const line = buffer.getLine(i);
            if (line) {
              lines.push(line.translateToString(true));
            }
          }
          if (lines.join('\n').trim()) {
            fullContext = `Terminal context:\n${lines.join('\n')}\n\n`;
          }
        }
      } else {
        // We have existing conversation history
        // Build context from conversation in chronological order
        fullContext = "Previous conversation:\n";
        aiState.conversationHistory.forEach(msg => {
          fullContext += `${msg.role}: ${msg.content}\n\n`;
        });
        
        // If we have additional context (selected text) for continuing conversation
        // We'll add it directly to the query that gets stored, not here
        // This prevents duplication since it will be in the User message
      }
      
      // Add current query with any selected text context
      if (initialContext && !isNewConversation) {
        fullContext += `\nAdditional terminal output to analyze:\n${initialContext}\n\n`;
      }
      fullContext += `Current question: ${query}`;
      
      // Create system prompt for terminal context
      const systemPrompt = `You are an AI assistant integrated directly into a terminal emulator. The user is viewing your response inline with their terminal session.

FORMATTING GUIDELINES:
1. Use Unicode icons liberally to enhance readability:
   - ✅ for success/correct  
   - ❌ for errors/incorrect
   - ⚠️ for warnings
   - 💡 for tips/suggestions
   - 📁 for directories
   - 📄 for files
   - 🔧 for configuration/settings
   - 🚀 for performance/speed
   - 🔒 for security/permissions
   - 📦 for packages/dependencies
   - 🐛 for bugs/debugging
   - ⚡ for quick tips
   - 📝 for notes/documentation
   - 🎯 for goals/targets
   - ⭐ for important points

2. Response length should match the complexity of the question:
   - Simple questions: 2-5 lines
   - Error explanations: Include full solution steps
   - Concept explanations: Provide sufficient detail for understanding
   - Scripts/code: Include complete, working examples
   - Don't artificially limit responses if more information is genuinely helpful

3. Terminal-optimized formatting:
   - Keep lines under 80-120 characters for readability
   - Use backticks for \`inline code\` and code blocks
   - Number multi-step instructions (1. 2. 3.)
   - Use bullet points with • or - for lists
   - Add blank lines sparingly for visual separation

4. Content priorities:
   - Lead with the direct answer or solution
   - Show exact commands to run
   - Explain errors with actionable fixes
   - Include examples when helpful
   - Add context only when it aids understanding

5. Be conversational but efficient:
   - Use icons to reduce text verbosity
   - Group related information visually
   - Highlight important warnings with ⚠️
   - Mark successful outcomes with ✅

Conversation context and question:
${fullContext}`;
      
      // Log the AI API call
      console.log('🤖 AI API Call:', {
        provider: $settings.aiProvider,
        currentQuery: query,
        conversationHistory: aiState.conversationHistory,
        historyLength: aiState.conversationHistory.length,
        fullContextLength: fullContext.length,
        fullPrompt: systemPrompt,
        timestamp: new Date().toISOString()
      });
      
      // Query the selected AI provider with system context
      const response = $settings.aiProvider === 'openrouter' 
        ? await openRouterService.queryOpenRouter(systemPrompt)
        : await geminiService.queryGemini(systemPrompt);
      
      // Log the AI response
      console.log('✅ AI Response:', {
        responseLength: response.length,
        responsePreview: response.substring(0, 200) + (response.length > 200 ? '...' : ''),
        timestamp: new Date().toISOString()
      });
      
      // Clear the loading message (only the loading line, keep the query visible)
      term.write('\x1b[1A\x1b[2K'); // Move up one line and clear it
      
      // Display the response with formatting
      term.write('\x1b[32m━━━ 🤖 AI ━━━\x1b[0m\r\n');
      
      // Format and display the response with improved markdown handling
      formatMarkdownForTerminal(response, term);
      
      term.write('\x1b[32m━━━━━━━━━━━━━\x1b[0m\r\n');
      
      // Add question and response to conversation history
      // Store them in chronological order in a natural, conversational way
      
      // Determine what the user actually communicated
      let userMessage = query;
      
      // Check if this is an internal prompt (when we're adding selected text)
      const isInternalPrompt = query.includes("Here's additional terminal output to add to our conversation") || 
                               query.includes("Analyze the following terminal output");
      
      if (initialContext && isInternalPrompt) {
        // User selected text and we're analyzing it - store what actually happened
        if (!isNewConversation) {
          userMessage = `Look at this additional output from my terminal:\n\n${initialContext}`;
        } else {
          userMessage = `Can you help me with this terminal output:\n\n${initialContext}`;
        }
      } else if (initialContext && !isInternalPrompt) {
        // User typed a question after selecting text (shouldn't happen with current flow)
        userMessage = `${initialContext}\n\n${query}`;
      }
      // Otherwise userMessage stays as the original query
      
      aiState.conversationHistory.push({role: 'User', content: userMessage});
      aiState.conversationHistory.push({role: 'Assistant', content: response});
      
      // Show context usage status
      const finalContextStatus = contextManager.getContextStatus(aiState.conversationHistory);
      const usageColor = finalContextStatus.percentageUsed > 80 ? '\x1b[33m' : '\x1b[36m';
      term.write(`${usageColor}📊  Context: ${Math.round(finalContextStatus.percentageUsed)}% used (${finalContextStatus.currentTokens}/${finalContextStatus.maxTokens} tokens)\x1b[0m\r\n`);
      
      // Show hint about AI mode with all available commands
      term.write('\x1b[90m(Commands: Enter to exit, /exit to leave, /new for fresh conversation)\x1b[0m\r\n');
      
    } catch (error) {
      // Clear the loading message (only the loading line, keep the query visible)
      term.write('\x1b[1A\x1b[2K'); // Move up one line and clear it
      
      const errorMessage = error instanceof Error ? error.message : 'Unknown error';
      term.write('\x1b[31m❌  AI Error: ' + errorMessage + '\x1b[0m\r\n');
      
      if (errorMessage.includes('API key')) {
        const providerName = $settings.aiProvider === 'openrouter' ? 'OpenRouter' : 'Gemini';
        term.write(`\x1b[33m🔑  Please configure your ${providerName} API key in Settings ⚙️\x1b[0m\r\n`);
      }
    } finally {
      aiState.isProcessingAI = false;
    }
  }

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

  // Rich export functionality
  async function handleRichExport(event: CustomEvent<{format: ExportFormat; options: ExportOptions}>) {
    if (!exportManager) {
      console.error('Export manager not initialized');
      return;
    }

    try {
      const { format, options } = event.detail;
      const result = await exportManager.export(options);
      exportManager.downloadExport(result);
      
      const formatName = format === 'zip' ? 'ZIP archive' : format.toUpperCase();
      makeToast({
        kind: "success",
        message: `Terminal exported as ${formatName}`,
      });
    } catch (error) {
      console.error('Rich export failed:', error);
      makeToast({
        kind: "error",
        message: `Export failed: ${error instanceof Error ? error.message : 'Unknown error'}`,
      });
    }
  }

  function updateExportManager() {
    if (!term || !loaded) return;

    const exportTheme = {
      background: theme.background || '#000000',
      foreground: theme.foreground || '#ffffff',
      cursor: theme.cursor || theme.foreground || '#ffffff',
      selection: 'rgba(255, 255, 255, 0.3)' // Standard selection color
    };

    const terminalInfo = {
      title: currentTitle,
      rows: rows,
      cols: cols,
      fontFamily: $settings.fontFamily,
      fontSize: $settings.fontSize
    };

    if (exportManager) {
      exportManager.updateContext(exportTheme, terminalInfo);
    } else {
      exportManager = new ExportManager(term, exportTheme, terminalInfo);
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
  
  getThumbnail = async () => {
    if (!term || !termEl) return null;
    
    console.log('🖼️ Starting terminal thumbnail capture...');
    
    try {
      // Force terminal to refresh/render before capture
      if (term) {
        term.refresh(0, term.rows - 1);
      }
      
      // Wait a frame for render to complete
      await new Promise(resolve => requestAnimationFrame(resolve));
      
      // Debug: List all canvas elements found
      const allCanvases = termEl.querySelectorAll('canvas');
      console.log(`📊 Found ${allCanvases.length} canvas elements:`, 
        Array.from(allCanvases).map(c => ({ 
          className: c.className, 
          width: c.width, 
          height: c.height,
          hasData: !isCanvasBlank(c)
        }))
      );
      
      // Method 1: Prioritize canvas elements with actual content, WebGL first
      const canvasElements = Array.from(allCanvases).map(canvas => {
        const canvasEl = canvas as HTMLCanvasElement;
        const isWebGL = canvasEl.getContext('webgl') || canvasEl.getContext('webgl2');
        const hasData = !isCanvasBlank(canvasEl);
        return {
          canvas: canvasEl,
          isWebGL,
          hasData,
          priority: isWebGL && hasData ? 1 : hasData ? 2 : isWebGL ? 3 : 4
        };
      }).sort((a, b) => a.priority - b.priority);
      
      console.log('📋 Canvas priority order:', canvasElements.map(c => 
        `${c.isWebGL ? 'WebGL' : '2D'} (${c.canvas.className || 'unnamed'}) - hasData: ${c.hasData}, priority: ${c.priority}`
      ));
      
      for (const {canvas: canvasEl, isWebGL} of canvasElements) {
        const canvasType = isWebGL ? 'WebGL' : '2D';
        const canvasDesc = `${canvasType} (${canvasEl.className || 'unnamed'})`;
        
        console.log(`🖼️ Attempting ${canvasDesc} capture...`);
        
        try {
          let thumbnail: string | null = null;
          
          if (isWebGL) {
            // Use WebGL-specific capture with timing
            thumbnail = await captureWebGLCanvasWithTiming(canvasEl);
          } else {
            // Use regular canvas capture
            thumbnail = captureRegularCanvas(canvasEl);
          }
          
          if (thumbnail && !isDataURLBlank(thumbnail)) {
            console.log(`✅ ${canvasDesc} capture successful!`);
            return thumbnail;
          }
          console.log(`❌ ${canvasDesc} capture returned blank/null`);
        } catch (error) {
          console.warn(`❌ ${canvasDesc} capture failed:`, error);
        }
      }
      
      // Method 3: DOM screenshot with html2canvas
      console.log('🌐 Attempting DOM screenshot with html2canvas...');
      try {
        const domThumbnail = await captureDOMScreenshot();
        if (domThumbnail && !isDataURLBlank(domThumbnail)) {
          console.log('✅ DOM screenshot successful');
          return domThumbnail;
        }
        console.log('❌ DOM screenshot returned blank/null');
      } catch (error) {
        console.warn('❌ DOM screenshot failed:', error);
      }
      
      // Method 4: Text-based fallback
      console.log('📝 Using text-based fallback...');
      const textPreview = createTextBasedPreview();
      console.log('✅ Text-based preview created');
      return textPreview;
      
    } catch (error) {
      console.warn('❌ Fatal error in thumbnail capture:', error);
      return createTextBasedPreview();
    }
  };
  
  getThumbnails = async () => {
    if (!term || !termEl) return {small: null, large: null};
    
    console.log('🖼️ Creating multiple thumbnail sizes...');
    
    try {
      // Get the best quality source
      let sourceDataURL: string | null = null;
      
      // Try canvas capture first
      const canvasElements = Array.from(termEl.querySelectorAll('canvas')).map(canvas => {
        const canvasEl = canvas as HTMLCanvasElement;
        const isWebGL = canvasEl.getContext('webgl') || canvasEl.getContext('webgl2');
        const hasData = !isCanvasBlank(canvasEl);
        return {
          canvas: canvasEl,
          isWebGL,
          hasData,
          priority: isWebGL && hasData ? 1 : hasData ? 2 : isWebGL ? 3 : 4
        };
      }).sort((a, b) => a.priority - b.priority);
      
      for (const {canvas: canvasEl, isWebGL} of canvasElements) {
        try {
          if (isWebGL) {
            sourceDataURL = await captureWebGLCanvasWithTiming(canvasEl);
          } else {
            sourceDataURL = captureRegularCanvas(canvasEl);
          }
          
          if (sourceDataURL && !isDataURLBlank(sourceDataURL)) {
            break; // Found good source
          }
        } catch (error) {
          console.warn('Canvas capture failed:', error);
        }
      }
      
      if (!sourceDataURL) {
        // Fallback to text-based
        const textPreview = createTextBasedPreview();
        return {
          small: textPreview,
          large: textPreview
        };
      }
      
      // Create image from source to scale from
      const sourceImg = new Image();
      await new Promise((resolve, reject) => {
        sourceImg.onload = resolve;
        sourceImg.onerror = reject;
        sourceImg.src = sourceDataURL!; // We checked for null above
      });
      
      // Create small thumbnail (for list)
      const smallThumbnail = createScaledThumbnailFromImage(sourceImg, 80, 48);
      
      // Use native resolution for large thumbnail (best quality)
      const largeThumbnail = sourceDataURL;
      
      return {
        small: smallThumbnail,
        large: largeThumbnail
      };
      
    } catch (error) {
      console.warn('Failed to create thumbnails:', error);
      const fallback = createTextBasedPreview();
      return {
        small: fallback,
        large: fallback
      };
    }
  };
  
  function createScaledThumbnailFromImage(sourceImg: HTMLImageElement, width: number, height: number): string | null {
    try {
      const devicePixelRatio = window.devicePixelRatio || 1;
      const scaledWidth = width * devicePixelRatio;
      const scaledHeight = height * devicePixelRatio;
      
      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');
      if (!ctx) return null;
      
      canvas.width = scaledWidth;
      canvas.height = scaledHeight;
      canvas.style.width = width + 'px';
      canvas.style.height = height + 'px';
      
      // High quality scaling
      ctx.imageSmoothingEnabled = true;
      ctx.imageSmoothingQuality = 'high';
      ctx.scale(devicePixelRatio, devicePixelRatio);
      
      // Draw scaled image
      ctx.drawImage(sourceImg, 0, 0, width, height);
      
      // Add border
      ctx.strokeStyle = theme.cursor || '#ffffff';
      ctx.lineWidth = 1;
      ctx.strokeRect(0, 0, width, height);
      
      return canvas.toDataURL('image/png', 1.0);
    } catch (error) {
      console.warn('Failed to scale image:', error);
      return null;
    }
  }
  
  function isCanvasBlank(canvas: HTMLCanvasElement): boolean {
    try {
      // Check if this is a WebGL canvas by testing for WebGL context
      const isWebGL = canvas.getContext('webgl') || canvas.getContext('webgl2');
      
      if (isWebGL) {
        // For WebGL canvases, use toDataURL method
        try {
          const dataURL = canvas.toDataURL();
          return isDataURLBlank(dataURL);
        } catch {
          return true; // If toDataURL fails, assume blank
        }
      }
      
      // For 2D canvases, use pixel data checking
      const ctx = canvas.getContext('2d');
      if (!ctx) return true;
      
      const imageData = ctx.getImageData(0, 0, Math.min(canvas.width, 50), Math.min(canvas.height, 50));
      const data = imageData.data;
      
      // Check if all pixels are transparent or black (sample just a small area for performance)
      for (let i = 0; i < data.length; i += 4) {
        if (data[i] !== 0 || data[i + 1] !== 0 || data[i + 2] !== 0 || data[i + 3] !== 0) {
          return false; // Found non-black/transparent pixel
        }
      }
      return true;
    } catch {
      return true; // Assume blank if we can't read
    }
  }
  
  function isDataURLBlank(dataURL: string): boolean {
    if (!dataURL || dataURL === '') return true;
    
    // Quick check for very small data URLs (likely blank)
    if (dataURL.length < 100) return true;
    
    // Check for common blank image patterns
    const blankPatterns = [
      'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==', // 1x1 transparent
      'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=' // 1x1 black
    ];
    
    for (const pattern of blankPatterns) {
      if (dataURL.startsWith(pattern.substring(0, 50))) {
        return true;
      }
    }
    
    // Check if the data URL is suspiciously small for the expected canvas size
    // A 300x180 PNG should be much larger than 1000 bytes when base64 encoded
    if (dataURL.length < 1000) {
      console.log('🚨 Data URL suspiciously small:', dataURL.length, 'bytes');
      return true;
    }
    
    return false;
  }
  
  async function captureWebGLCanvasWithTiming(canvas: HTMLCanvasElement): Promise<string | null> {
    try {
      console.log('🎮 WebGL canvas details:', {
        width: canvas.width,
        height: canvas.height,
        context: canvas.getContext('webgl') ? 'webgl' : canvas.getContext('webgl2') ? 'webgl2' : 'none'
      });
      
      // Try to force a WebGL render by requesting animation frame multiple times
      for (let i = 0; i < 3; i++) {
        await new Promise(resolve => requestAnimationFrame(resolve));
      }
      
      // Check if canvas has any content before trying to capture
      if (isCanvasBlank(canvas)) {
        console.log('❌ WebGL canvas appears blank before capture');
        return null;
      }
      
      return createScaledThumbnail(canvas, 300, 180);
    } catch (error) {
      console.warn('❌ WebGL timing capture failed:', error);
      return null;
    }
  }
  
  function captureWebGLCanvas(canvas: HTMLCanvasElement): string | null {
    try {
      // For WebGL canvases, we need to ensure preserveDrawingBuffer was enabled
      // and capture immediately after a frame is rendered
      return createScaledThumbnail(canvas, 300, 180);
    } catch (error) {
      console.warn('WebGL canvas toDataURL failed:', error);
      return null;
    }
  }
  
  function captureRegularCanvas(canvas: HTMLCanvasElement): string | null {
    try {
      // Regular 2D canvas capture should work reliably
      return createScaledThumbnail(canvas, 300, 180);
    } catch (error) {
      console.warn('Regular canvas toDataURL failed:', error);
      return null;
    }
  }
  
  function createScaledThumbnail(sourceCanvas: HTMLCanvasElement, width: number, height: number): string | null {
    try {
      // Create a higher quality thumbnail canvas with device pixel ratio
      const devicePixelRatio = window.devicePixelRatio || 1;
      const scaledWidth = width * devicePixelRatio;
      const scaledHeight = height * devicePixelRatio;
      
      const thumbnailCanvas = document.createElement('canvas');
      const ctx = thumbnailCanvas.getContext('2d');
      if (!ctx) return null;
      
      // Set actual canvas size (higher res for quality)
      thumbnailCanvas.width = scaledWidth;
      thumbnailCanvas.height = scaledHeight;
      
      // Scale back down for CSS display
      thumbnailCanvas.style.width = width + 'px';
      thumbnailCanvas.style.height = height + 'px';
      
      // Enable high-quality image smoothing
      ctx.imageSmoothingEnabled = true;
      ctx.imageSmoothingQuality = 'high';
      
      // Scale the context to match device pixel ratio
      ctx.scale(devicePixelRatio, devicePixelRatio);
      
      // Draw with high quality scaling
      ctx.drawImage(sourceCanvas, 0, 0, sourceCanvas.width, sourceCanvas.height, 0, 0, width, height);
      
      // Add a subtle border for consistency
      ctx.strokeStyle = theme.cursor || '#ffffff';
      ctx.lineWidth = 1;
      ctx.strokeRect(0, 0, width, height);
      
      // Use maximum quality for PNG output
      return thumbnailCanvas.toDataURL('image/png', 1.0);
    } catch (error) {
      console.warn('Failed to create scaled thumbnail:', error);
      return null;
    }
  }
  
  async function captureDOMScreenshot(): Promise<string | null> {
    try {
      // Try to capture terminal DOM content
      console.log('🌐 Attempting html2canvas capture...');
      
      // First try to find the terminal viewport/screen
      const terminalViewport = termEl.querySelector('.xterm-viewport') as HTMLElement;
      const terminalScreen = termEl.querySelector('.xterm-screen') as HTMLElement;
      const terminalRows = termEl.querySelector('.xterm-rows') as HTMLElement;
      
      // Prefer the most specific element
      const targetElement = terminalRows || terminalScreen || terminalViewport || termEl;
      
      console.log('🎯 Capturing element:', {
        className: targetElement.className,
        width: targetElement.offsetWidth,
        height: targetElement.offsetHeight,
        hasChildren: targetElement.children.length
      });
      
      // Use html2canvas with optimized settings for terminal content
      const html2canvas = (await import('html2canvas')).default;
      
      const canvas = await html2canvas(targetElement, {
        backgroundColor: theme.background || '#000000',
        scale: 1, // Use full scale first
        useCORS: true,
        allowTaint: true,
        foreignObjectRendering: false, // Disable for better terminal compatibility
        imageTimeout: 1000,
        removeContainer: false,
        logging: false,
        onclone: (clonedDoc, element) => {
          // Ensure the cloned element maintains visibility
          element.style.display = 'block';
          element.style.visibility = 'visible';
          element.style.opacity = '1';
          return element;
        }
      });
      
      console.log('📏 html2canvas result:', { width: canvas.width, height: canvas.height });
      
      // Create thumbnail from the captured canvas
      const thumbnail = createScaledThumbnail(canvas, 300, 180);
      if (thumbnail && !isDataURLBlank(thumbnail)) {
        return thumbnail;
      }
      
      console.log('❌ html2canvas produced blank result, trying manual DOM render...');
      return captureDOMManualRender();
      
    } catch (error) {
      console.warn('❌ html2canvas capture failed:', error);
      return captureDOMManualRender();
    }
  }
  
  function captureDOMManualRender(): string | null {
    try {
      console.log('🎨 Attempting manual DOM render...');
      
      // Try to extract text content from terminal rows
      const rowElements = termEl.querySelectorAll('.xterm-rows .xterm-row');
      
      console.log(`📝 Found ${rowElements.length} terminal rows`);
      
      if (rowElements.length === 0) {
        console.log('❌ No terminal rows found, using fallback');
        return captureDOMScreenshotFallback();
      }
      
      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');
      if (!ctx) return null;
      
      canvas.width = 300;
      canvas.height = 180;
      
      // Fill background
      ctx.fillStyle = theme.background || '#000000';
      ctx.fillRect(0, 0, canvas.width, canvas.height);
      
      // Set text properties
      ctx.font = '9px monospace';
      ctx.fillStyle = theme.foreground || '#ffffff';
      
      const lineHeight = 12;
      let yOffset = lineHeight;
      
      // Render visible terminal content
      const maxRows = Math.min(14, rowElements.length);
      const startRow = Math.max(0, rowElements.length - maxRows);
      
      for (let i = startRow; i < rowElements.length && yOffset < canvas.height - 5; i++) {
        const row = rowElements[i];
        let text = '';
        
        // Extract text from the row, handling spans and other elements
        const textContent = row.textContent || (row as HTMLElement).innerText || '';
        text = textContent.trim();
        
        if (text) {
          // Truncate if too long
          if (text.length > 45) {
            text = text.substring(0, 42) + '...';
          }
          
          ctx.fillText(text, 5, yOffset);
        }
        
        yOffset += lineHeight;
      }
      
      // Add border
      ctx.strokeStyle = theme.cursor || '#ffffff';
      ctx.lineWidth = 1;
      ctx.strokeRect(0, 0, canvas.width, canvas.height);
      
      console.log('✅ Manual DOM render complete');
      return canvas.toDataURL('image/png', 0.9);
      
    } catch (error) {
      console.warn('❌ Manual DOM render failed:', error);
      return captureDOMScreenshotFallback();
    }
  }
  
  function captureDOMScreenshotFallback(): string | null {
    try {
      // Simplified DOM capture fallback
      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');
      if (!ctx) return null;
      
      canvas.width = 300;
      canvas.height = 180;
      
      // Fill with terminal background
      ctx.fillStyle = theme.background || '#000000';
      ctx.fillRect(0, 0, canvas.width, canvas.height);
      
      // Add some indication this is a fallback capture
      ctx.fillStyle = theme.foreground || '#ffffff';
      ctx.font = '12px monospace';
      ctx.fillText('Terminal View', 10, 30);
      ctx.fillText('(Fallback Capture)', 10, 50);
      
      // Add terminal border
      ctx.strokeStyle = theme.cursor || '#ffffff';
      ctx.lineWidth = 1;
      ctx.strokeRect(0, 0, canvas.width, canvas.height);
      
      return canvas.toDataURL('image/png', 0.9);
    } catch (error) {
      console.warn('Fallback DOM screenshot failed:', error);
      return null;
    }
  }
  
  function createTextBasedPreview(): string | null {
    try {
      // Fallback to the original text-based approach
      const buffer = term!.buffer.active;
      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');
      if (!ctx) return null;
      
      // Set thumbnail size
      canvas.width = 300;
      canvas.height = 180;
      
      // Fill background with terminal background color
      ctx.fillStyle = theme.background || '#000000';
      ctx.fillRect(0, 0, canvas.width, canvas.height);
      
      // Set text properties
      ctx.font = '10px monospace';
      ctx.fillStyle = theme.foreground || '#ffffff';
      
      // Get visible lines from buffer (last 12 lines)
      const lineHeight = 14;
      const maxLines = Math.min(12, buffer.length);
      const startLine = Math.max(0, buffer.cursorY - maxLines + 1);
      
      let yOffset = 10;
      for (let i = startLine; i <= buffer.cursorY && i < buffer.length; i++) {
        const line = buffer.getLine(i);
        if (line) {
          const text = line.translateToString(true);
          // Truncate long lines
          const truncated = text.length > 40 ? text.substring(0, 40) + '...' : text;
          ctx.fillText(truncated, 5, yOffset);
          yOffset += lineHeight;
          
          if (yOffset > canvas.height - 10) break;
        }
      }
      
      // Add a subtle border
      ctx.strokeStyle = theme.cursor || '#ffffff';
      ctx.lineWidth = 1;
      ctx.strokeRect(0, 0, canvas.width, canvas.height);
      
      // Return as data URL
      return canvas.toDataURL('image/png', 0.9);
    } catch (error) {
      console.warn('Failed to create text-based preview:', error);
      return null;
    }
  }

  $: if (term) {
    term.resize(cols, rows);
    // If we're in AI mode, restore the AI prompt after resize
    if (aiState.isInAIMode) {
      // Use a small delay to let the resize complete
      setTimeout(() => {
        if (term && aiState.isInAIMode) {
          // Clear any shell prompt that might have appeared
          term.write('\r\x1b[K'); // Move to start of line and clear it
          // Restore AI prompt with current buffer
          term.write('\x1b[36m✨  \x1b[0m' + aiState.aiCommandBuffer);
        }
      }, 10);
    }
  }

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
      fontWeight: $settings.fontWeight,
      fontWeightBold: $settings.fontWeightBold,
      lineHeight: 1.06,
      scrollback: $settings.scrollback,
      theme,
    });

    // Keyboard shortcuts for natural text editing.
    term.attachCustomKeyEventHandler((event) => {
      // Ctrl+Shift+A or Cmd+Shift+A for AI query
      if (
        ((isMac && event.metaKey) || (!isMac && event.ctrlKey)) &&
        event.shiftKey && 
        event.key === 'A' &&
        !event.altKey &&
        term
      ) {
        event.preventDefault();
        const selection = term.getSelection();
        if (selection) {
          // Enter AI mode with selected text
          aiState.isInAIMode = true;
          // Don't reset conversation - user might want to add this to existing context
          
          // Move to new line if needed
          const buffer = term.buffer.active;
          if (buffer.cursorX > 0) {
            term.write('\r\n');
          }
          
          const hasHistory = aiState.conversationHistory.length > 0;
          term.write('\x1b[32m━━━ AI Mode ━━━\x1b[0m\r\n');
          if (hasHistory) {
            const contextStatus = contextManager.getContextStatus(aiState.conversationHistory);
            term.write(`\x1b[36m📚  Continuing previous conversation (${aiState.conversationHistory.filter(e => e.role !== 'Context').length / 2} exchanges, ${Math.round(contextStatus.percentageUsed)}% context used)\x1b[0m\r\n`);
            term.write('\x1b[33m📎  Adding selected text to conversation context...\x1b[0m\r\n');
          }
          term.write('\x1b[90m(Commands: Enter to exit, /exit to leave, /new for fresh conversation)\x1b[0m\r\n\r\n');
          
          // Adapt the prompt based on whether we have history
          const contextPrompt = hasHistory 
            ? `Here's additional terminal output to add to our conversation context. Analyze it and provide the most helpful response based on our previous discussion and this new information.`
            : `Analyze the following terminal output and provide the most helpful response. If it's an error, explain how to fix it. If it's command output, explain what it means. If it's code, explain what it does. Be helpful and contextual.`;
          
          processAIQuery(contextPrompt, selection, false).then(() => {
            if (term) term.write('\x1b[36m✨  \x1b[0m');
          });
        } else {
          // No selection, show help
          term.write('\r\n\x1b[33m⚠️  Select text first, then press Ctrl+Shift+A (or Cmd+Shift+A) to ask AI\x1b[0m\r\n');
        }
        return false;
      }
      
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
    // Enable preserveDrawingBuffer to allow screenshot capture of WebGL canvas
    term.loadAddon(new WebglAddon(true)); // preserveDrawingBuffer: true
    term.loadAddon(new ImageAddon({ enableSizeReports: false }));

    term.open(termEl);

    term.resize(cols, rows);
    term.onTitleChange((title) => {
      currentTitle = title;
      dispatch('titleChange', title);
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
    
    // Initialize export manager after terminal is loaded
    updateExportManager();

    const utf8 = new TextEncoder();
    
    term.onData(async (data: string) => {
      // If we're in AI mode, handle everything as AI input
      if (aiState.isInAIMode) {
        if (data === '\r' || data === '\n') {
          const query = aiState.aiCommandBuffer.trim();
          
          // Check for special commands
          if (query === '/new') {
            // Clear conversation history explicitly
            aiState.conversationHistory = [];
            aiState.aiCommandBuffer = "";
            
            // Clear the input line
            const clearLength = 4; // Length of "/new"
            let clearSequence = '';
            for (let i = 0; i < clearLength; i++) {
              clearSequence += '\b \b';
            }
            if (term) {
              term.write(clearSequence);
              term.write('\r\n');
              term.write('\x1b[32m🔄 Started new conversation\x1b[0m\r\n');
              term.write('\x1b[36m✨  \x1b[0m');
            }
          } else if (query === '' || query === '/exit') {
            // Clear the input line if we typed /exit
            if (query === '/exit') {
              const clearLength = aiState.aiCommandBuffer.length;
              let clearSequence = '';
              for (let i = 0; i < clearLength; i++) {
                clearSequence += '\b \b';
              }
              if (term) term.write(clearSequence);
            }
            
            // Exit AI mode and return to shell (but keep conversation)
            if (term) {
              term.write('\r\n');
              term.write('\x1b[90mExiting AI mode (conversation preserved)...\x1b[0m\r\n');
            }
            aiState.isInAIMode = false;
            aiState.aiCommandBuffer = "";
            // Don't clear conversation history - keep it for next time
            // Send a single Enter to restore shell prompt
            dispatch("data", utf8.encode('\r'));
          } else {
            // Move to new line, showing the user's input as "committed"
            if (term) term.write('\r\n');
            
            // Reset buffer immediately so it doesn't appear duplicated
            aiState.aiCommandBuffer = "";
            
            // Process the query - don't show query again since user already saw it
            await processAIQuery(query, "", false);
            
            // Show AI prompt with sparkle and space after response
            if (term) term.write('\x1b[36m✨  \x1b[0m');
          }
        } else if (data === '\x7f') { // Backspace
          if (aiState.aiCommandBuffer.length > 0) {
            aiState.aiCommandBuffer = aiState.aiCommandBuffer.slice(0, -1);
            if (term) term.write('\b \b');
          }
        } else if (data === '\x03') { // Ctrl+C - exit AI mode
          if (term) {
            term.write('\r\n');
            term.write('\x1b[90mExiting AI mode (conversation preserved)...\x1b[0m\r\n');
          }
          aiState.isInAIMode = false;
          aiState.aiCommandBuffer = "";
          // Don't clear conversation history - keep it for next time
          // Don't send anything - user will get prompt when they type
        } else if (data.charCodeAt(0) >= 32) { // Printable characters
          aiState.aiCommandBuffer += data;
          if (term) term.write(data);
        }
        return; // Don't process further in AI mode
      }
      
      // Normal mode - just pass everything through to shell
      dispatch("data", utf8.encode(data));
    });
    term.onBinary((data: string) => {
      dispatch("data", Buffer.from(data, "binary"));
    });

    // Add copy-on-select functionality
    let selectionTimer: number | null = null;
    
    if (term) {
      term.onSelectionChange(() => {
        const selection = term?.getSelection();
        
        // Track selected text for AI sparkles button
        aiState.selectedTextForAI = selection || "";
      
      // Handle copy-on-select
      if (!$settings.copyOnSelect) return;
      
      // Clear any existing timer
      if (selectionTimer !== null) {
        clearTimeout(selectionTimer);
      }
      
      // Set a small delay to ensure selection is complete
      selectionTimer = setTimeout(() => {
        if (selection) {
          copyToClipboard(selection);
        }
        selectionTimer = null;
      }, 50) as any;
      });
    }

    // Add middle-click paste functionality
    termEl.addEventListener('mouseup', async (event: MouseEvent) => {
      // Check if middle mouse button (button 1) was clicked
      if (event.button === 1 && $settings.middleClickPaste) {
        event.preventDefault();
        event.stopPropagation();
        
        try {
          const text = await navigator.clipboard.readText();
          if (text) {
            // Send the pasted text as input data
            const utf8 = new TextEncoder();
            dispatch("data", utf8.encode(text));
          }
        } catch (err) {
          console.error('Failed to read clipboard for middle-click paste:', err);
        }
      }
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
    <div class="flex-1 flex items-center justify-end gap-2 px-3">
      {#if $settings.aiEnabled && (($settings.aiProvider === 'gemini' && $settings.geminiApiKey) || ($settings.aiProvider === 'openrouter' && $settings.openRouterApiKey))}
        {#if aiState.conversationHistory.length > 0 && !aiState.isInAIMode}
          <!-- Continue conversation button (only visible when conversation exists) -->
          <button
            class="ai-titlebar-button continue-conversation"
            class:has-selection={aiState.selectedTextForAI}
            title={aiState.selectedTextForAI ? "Add selected text to conversation" : "Continue AI conversation"}
            on:mousedown={async (event) => {
              if (event.button === 0) {
                event.stopPropagation();
                if (term) {
                  // Move to new line if not at start of line
                  const buffer = term.buffer.active;
                  if (buffer.cursorX > 0) {
                    term.write('\r\n');
                  }
                  
                  // Enter AI mode - continue existing conversation
                  aiState.isInAIMode = true;
                  term.write('\x1b[32m━━━ Continuing AI Conversation ━━━\x1b[0m\r\n');
                  const contextStatus = contextManager.getContextStatus(aiState.conversationHistory);
                  term.write(`\x1b[36m📚 Resuming conversation (${aiState.conversationHistory.filter(e => e.role !== 'Context').length / 2} exchanges, ${Math.round(contextStatus.percentageUsed)}% context used)\x1b[0m\r\n`);
                  
                  // If text is selected, add it to the conversation
                  if (aiState.selectedTextForAI) {
                    term.write('\x1b[33m📎 Adding selected text to conversation context...\x1b[0m\r\n');
                  }
                  
                  term.write('\x1b[90m(Commands: /exit to leave, /new to start fresh conversation)\x1b[0m\r\n\r\n');
                  
                  // If text is selected, process it
                  if (aiState.selectedTextForAI) {
                    const contextPrompt = `Here's additional terminal output to add to our conversation context. Analyze it and provide the most helpful response based on our previous discussion and this new information.`;
                    // Don't show the internal prompt, just process with context
                    await processAIQuery(contextPrompt, aiState.selectedTextForAI, false);
                    
                    // Clear selection after processing
                    term.clearSelection();
                    aiState.selectedTextForAI = "";
                  }
                  
                  // Show AI prompt with sparkle and space
                  term.write('\x1b[36m✨  \x1b[0m');
                  
                  // Reset state
                  aiState.aiCommandBuffer = "";
                }
              }
            }}
          >
            <SparklesIcon size={14} />
          </button>
        {/if}
        
        <!-- New conversation button (always visible when AI is enabled) -->
        <button
          class="ai-titlebar-button new-conversation"
          class:has-selection={aiState.selectedTextForAI}
          title={aiState.selectedTextForAI ? "Start new conversation with selected text" : "Start new AI conversation"}
          on:mousedown={async (event) => {
            if (event.button === 0) {
              event.stopPropagation();
              if (term) {
                // Clear conversation history for new conversation
                aiState.conversationHistory = [];
                
                // Move to new line if not at start of line
                const buffer = term.buffer.active;
                if (buffer.cursorX > 0) {
                  term.write('\r\n');
                }
                
                // Enter AI mode with fresh conversation
                aiState.isInAIMode = true;
                term.write('\x1b[32m━━━ Starting New AI Conversation ━━━\x1b[0m\r\n');
                term.write('\x1b[90m(Commands: /exit to leave, /new to start fresh conversation)\x1b[0m\r\n\r\n');
                
                // If text is selected, process it
                if (aiState.selectedTextForAI) {
                  // Let the AI intelligently respond based on the context
                  const contextPrompt = `Analyze the following terminal output and provide the most helpful response. If it's an error, explain how to fix it. If it's command output, explain what it means. If it's code, explain what it does. Be helpful and contextual.`;
                  // Don't show the internal prompt, just process with context
                  await processAIQuery(contextPrompt, aiState.selectedTextForAI, false);
                  
                  // Clear selection after processing
                  term.clearSelection();
                  aiState.selectedTextForAI = "";
                }
                
                // Show AI prompt with sparkle and space
                term.write('\x1b[36m✨  \x1b[0m');
                
                // Reset state
                aiState.aiCommandBuffer = "";
              }
            }
          }}
        >
          <SparklesNewIcon size={14} />
        </button>
      {/if}
      {#if $settings.copyButtonEnabled}
        <button
          class="w-4 h-4 p-0.5 rounded hover:bg-theme-bg-tertiary transition-colors"
          title={copyButtonTooltip}
          on:mousedown={(event) => {
            if (event.button === 0) {
              event.stopPropagation();
              copyContentToClipboard();
            }
          }}
        >
          <svelte:component 
            this={copyButtonIcon}
            class="w-full h-full text-theme-fg-secondary"
            strokeWidth={2}
          />
        </button>
      {/if}
      <div class="relative">
        <button
          class="w-4 h-4 p-0.5 rounded hover:bg-theme-bg-tertiary transition-colors"
          title="Export terminal session"
          on:mousedown={(event) => {
            if (event.button === 0) {
              event.stopPropagation();
              showExportModal = true;
            }
          }}
        >
          <DownloadIcon
            class="w-full h-full text-theme-fg-secondary"
            strokeWidth={2}
          />
        </button>
        
        <ExportModal
          bind:open={showExportModal}
          hasSelection={term?.hasSelection() || false}
          terminalTitle={currentTitle}
          on:export={handleRichExport}
          on:close={() => showExportModal = false}
        />
      </div>
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
    position: relative;
  }

  .term-container:not(.focused) :global(.xterm) {
    @apply cursor-default;
  }

  .term-container.focused {
    @apply opacity-100;
  }
  
  .ai-titlebar-button {
    @apply p-0.5 rounded transition-all flex items-center justify-center;
    width: 20px;
    height: 20px;
  }
  
  /* Continue conversation button - blue */
  .ai-titlebar-button.continue-conversation {
    background: rgba(59, 130, 246, 0.1);
    color: #3B82F6;
  }
  
  .ai-titlebar-button.continue-conversation:hover {
    background: rgba(59, 130, 246, 0.2);
  }
  
  .ai-titlebar-button.continue-conversation.has-selection {
    background: rgba(59, 130, 246, 0.25);
    color: #2563EB;
    box-shadow: 0 0 4px rgba(59, 130, 246, 0.3);
  }
  
  .ai-titlebar-button.continue-conversation.has-selection:hover {
    background: rgba(59, 130, 246, 0.35);
  }
  
  .ai-titlebar-button.new-conversation {
    background: rgba(255, 165, 0, 0.1);
    color: #FFA500;
  }
  
  .ai-titlebar-button.new-conversation:hover {
    background: rgba(255, 165, 0, 0.2);
  }
  
  .ai-titlebar-button.new-conversation.has-selection {
    background: rgba(255, 140, 0, 0.25);
    color: #FF8C00;
    box-shadow: 0 0 4px rgba(255, 165, 0, 0.3);
  }
  
  .ai-titlebar-button.new-conversation.has-selection:hover {
    background: rgba(255, 140, 0, 0.35);
  }
</style>
