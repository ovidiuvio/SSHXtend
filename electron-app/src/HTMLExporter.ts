import type { Terminal } from '@xterm/xterm';
import type { SerializeAddon } from '@xterm/addon-serialize';

export interface ExportTheme {
  background: string;
  foreground: string;
  cursor?: string;
  selection?: string;
}

export interface TerminalInfo {
  fontFamily: string;
  fontSize: number;
  title?: string;
}

export class HTMLExporter {
  constructor(
    private term: Terminal,
    private serializeAddon: SerializeAddon,
    private theme: ExportTheme,
    private terminalInfo: TerminalInfo
  ) {}

  exportHTML(options: { selectionOnly?: boolean, optimizeForVSCode?: boolean, includeTimestamp?: boolean, title?: string } = {}): string {
    const htmlContent = this.serializeAddon.serializeAsHTML({
      onlySelection: options.selectionOnly || false,
      includeGlobalBackground: true
    });

    return this.wrapWithTemplate(htmlContent, options);
  }

  private wrapWithTemplate(content: string, options: any): string {
    const timestamp = new Date().toISOString();
    const title = options.title || this.terminalInfo.title || 'Terminal Session';

    return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>${this.escapeHtml(title)} - Local Terminal Export</title>
  <meta name="generator" content="SSHXtend Local Terminal">
  <meta name="export-timestamp" content="${timestamp}">
  <meta name="export-source" content="SSHXtend Terminal">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  ${this.getStyles(options)}
</head>
<body>
  <div class="terminal-export-container">
    ${options.includeTimestamp !== false ? this.getHeader(title, timestamp) : ''}
    <div class="terminal-content ${options.optimizeForVSCode ? 'vscode-optimized' : ''}">
      <pre class="terminal-output">${content}</pre>
    </div>
    ${this.getFooter()}
  </div>
</body>
</html>`;
  }

  private getStyles(options: any): string {
    const vsCodeOptimizations = options.optimizeForVSCode ? `
    /* VS Code compatibility variables */
    :root {
      --vscode-terminal-foreground: ${this.theme.foreground};
      --vscode-terminal-background: ${this.theme.background};
      --vscode-terminal-cursor: ${this.theme.cursor || this.theme.foreground};
      --vscode-terminal-selection: ${this.theme.selection || 'rgba(255, 255, 255, 0.3)'};
    }

    /* VS Code detection class */
    .vscode-optimized {
      /* Optimized for VS Code HTML preview */
      font-feature-settings: "liga" 0, "calt" 0;
      text-rendering: optimizeSpeed;
    }

    /* VS Code live preview compatibility */
    @media (prefers-color-scheme: dark) {
      body:not(.light-theme) {
        background-color: var(--vscode-terminal-background);
        color: var(--vscode-terminal-foreground);
      }
    }` : '';

    return `<style>
    /* Terminal export styles */
    * {
      box-sizing: border-box;
    }

    body {
      margin: 0;
      padding: 20px;
      font-family: ${this.terminalInfo.fontFamily}, 'Cascadia Code', 'Consolas', 'Monaco', 'Courier New', monospace;
      font-size: ${this.terminalInfo.fontSize}px;
      line-height: 1.2;
      background-color: ${this.theme.background};
      color: ${this.theme.foreground};
      overflow-x: auto;
    }

    .terminal-export-container {
      max-width: 100%;
      margin: 0 auto;
    }

    .terminal-header {
      background: rgba(255, 255, 255, 0.05);
      border: 1px solid rgba(255, 255, 255, 0.1);
      border-radius: 6px 6px 0 0;
      padding: 12px 16px;
      border-bottom: none;
      display: flex;
      justify-content: space-between;
      align-items: center;
      font-size: ${Math.max(12, this.terminalInfo.fontSize - 2)}px;
      color: rgba(${this.hexToRgb(this.theme.foreground)}, 0.8);
    }

    .terminal-title {
      font-weight: 600;
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .terminal-title::before {
      content: "🖥️";
      font-size: 16px;
    }

    .terminal-timestamp {
      font-family: monospace;
      opacity: 0.7;
      font-size: 11px;
    }

    .terminal-content {
      background-color: ${this.theme.background};
      border: 1px solid rgba(255, 255, 255, 0.1);
      border-radius: 0 0 6px 6px;
      overflow: auto;
    }

    .terminal-content.has-header {
      border-top: none;
      border-radius: 0 0 6px 6px;
    }

    .terminal-content:not(.has-header) {
      border-radius: 6px;
    }

    .terminal-output {
      margin: 0;
      padding: 16px;
      white-space: pre-wrap;
      word-wrap: break-word;
      font-family: inherit;
      font-size: inherit;
      line-height: inherit;
      background: transparent;
      overflow-x: auto;
    }

    .terminal-footer {
      margin-top: 16px;
      text-align: center;
      font-size: 11px;
      opacity: 0.6;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    }

    .terminal-footer a {
      color: #60a5fa;
      text-decoration: none;
    }

    .terminal-footer a:hover {
      text-decoration: underline;
    }

    /* Print styles */
    @media print {
      body {
        padding: 10px;
      }
      .terminal-footer {
        display: none;
      }
    }

    /* Mobile responsiveness */
    @media (max-width: 768px) {
      body {
        padding: 10px;
        font-size: ${Math.max(12, this.terminalInfo.fontSize - 2)}px;
      }

      .terminal-header {
        padding: 8px 12px;
        flex-direction: column;
        align-items: flex-start;
        gap: 4px;
      }

      .terminal-output {
        padding: 12px;
      }
    }

    /* Selection styling */
    ::selection {
      background-color: ${this.theme.selection || 'rgba(255, 255, 255, 0.3)'};
    }

    ::-moz-selection {
      background-color: ${this.theme.selection || 'rgba(255, 255, 255, 0.3)'};
    }

    ${vsCodeOptimizations}
  </style>`;
  }

  private getHeader(title: string, timestamp: string): string {
    const formattedTime = new Date(timestamp).toLocaleString();
    return `
    <div class="terminal-header">
      <div class="terminal-title">${this.escapeHtml(title)}</div>
      <div class="terminal-timestamp">${formattedTime}</div>
    </div>`;
  }

  private getFooter(): string {
    return `
    <div class="terminal-footer">
      Exported from SSHXtend Local Terminal
    </div>`;
  }

  private escapeHtml(text: string): string {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  private hexToRgb(hex: string): string {
    const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
    if (!result) return '255, 255, 255';

    return `${parseInt(result[1], 16)}, ${parseInt(result[2], 16)}, ${parseInt(result[3], 16)}`;
  }
}
