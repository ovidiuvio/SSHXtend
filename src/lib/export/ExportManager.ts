// Terminal export manager - coordinates all export formats
import type { Terminal } from 'sshx-xterm';
import JSZip from 'jszip';
import { HTMLExporter } from './HTMLExporter';
import { ANSIExporter } from './ANSIExporter';
import { MarkdownExporter } from './MarkdownExporter';
import type { ExportFormat, ExportOptions, ExportResult, ExportTheme, TerminalInfo } from './types';

export class ExportManager {
  private htmlExporter: HTMLExporter;
  private ansiExporter: ANSIExporter;
  private markdownExporter: MarkdownExporter;

  constructor(
    private term: Terminal,
    private theme: ExportTheme,
    private terminalInfo: TerminalInfo
  ) {
    this.htmlExporter = new HTMLExporter(term, theme, terminalInfo);
    this.ansiExporter = new ANSIExporter(term, terminalInfo);
    this.markdownExporter = new MarkdownExporter(term, terminalInfo);
  }

  /**
   * Export terminal content in the specified format
   */
  export(options: ExportOptions): ExportResult | Promise<ExportResult> {
    switch (options.format) {
      case 'html':
        return this.htmlExporter.export(options);
      case 'ansi':
        return this.ansiExporter.export(options);
      case 'markdown':
        return this.markdownExporter.export(options);
      case 'txt':
        return this.exportPlainText(options);
      case 'zip':
        return this.exportAllFormats(options);
      default:
        throw new Error(`Unsupported export format: ${options.format}`);
    }
  }

  /**
   * Quick plain text export (preserves current functionality)
   */
  exportPlainText(options: ExportOptions): ExportResult {
    let content: string;

    if (options.selectionOnly && this.term.hasSelection()) {
      content = this.term.getSelection() || '';
    } else {
      // Use existing logic from downloadTerminalText
      content = this.extractPlainTextContent();
    }

    const filename = this.generatePlainTextFilename(options);

    return {
      content,
      filename,
      mimeType: 'text/plain'
    };
  }

  /**
   * Export all formats as a zip archive
   */
  async exportAllFormats(options: ExportOptions): Promise<ExportResult> {
    const zip = new JSZip();
    const timestamp = new Date().toISOString().slice(0, 19).replace(/:/g, "-");
    const title = (options.title || this.terminalInfo.title || 'terminal')
      .replace(/[^a-zA-Z0-9]/g, "_")
      .toLowerCase();
    const suffix = options.selectionOnly ? 'selection' : 'session';

    // Export each format and add to zip
    const formats: ExportFormat[] = ['html', 'ansi', 'markdown', 'txt'];
    
    for (const format of formats) {
      const formatOptions = { ...options, format };
      const result = this.exportSingleFormat(formatOptions);
      zip.file(result.filename, result.content);
    }

    // Add README explaining the formats
    const readme = this.generateZipReadme(options);
    zip.file('README.txt', readme);

    // Generate zip content asynchronously
    const zipContent = await zip.generateAsync({ 
      type: 'uint8array',
      compression: 'DEFLATE',
      compressionOptions: { level: 6 }
    });

    return {
      content: zipContent as any, // Will be handled specially in downloadExport
      filename: `terminal-${title}-${suffix}-${timestamp}.zip`,
      mimeType: 'application/zip'
    };
  }

  /**
   * Export a single format (helper for zip export)
   */
  private exportSingleFormat(options: ExportOptions): ExportResult {
    switch (options.format) {
      case 'html':
        return this.htmlExporter.export(options);
      case 'ansi':
        return this.ansiExporter.export(options);
      case 'markdown':
        return this.markdownExporter.export(options);
      case 'txt':
        return this.exportPlainText(options);
      default:
        throw new Error(`Unsupported export format: ${options.format}`);
    }
  }

  /**
   * Generate README for zip archive
   */
  private generateZipReadme(options: ExportOptions): string {
    const title = options.title || this.terminalInfo.title || 'Terminal Session';
    const timestamp = new Date().toISOString();
    
    return `# ${title} - Export Archive

Exported: ${new Date(timestamp).toLocaleString()}
Source: SSHXtend Terminal Export
Terminal Size: ${this.terminalInfo.cols}×${this.terminalInfo.rows}
Font: ${this.terminalInfo.fontFamily} (${this.terminalInfo.fontSize}px)
${options.selectionOnly ? 'Content: Selected text only' : 'Content: Full terminal session'}

## Files Included

### 📄 .txt - Plain Text
- Simple text format without formatting
- Universal compatibility
- Smallest file size

### 🌐 .html - HTML Format (Recommended for VS Code)
- Perfect color and formatting preservation
- Opens directly in VS Code with full styling
- CSS optimized for VS Code display
- Mobile responsive design

### 🎨 .ansi - ANSI Format
- Raw terminal escape sequences preserved
- View with: cat filename.ansi
- Compatible with any terminal emulator
- Native terminal color support

### 📝 .md - Markdown Format
- Documentation-friendly format
- GitHub-compatible rendering
- Syntax highlighting in code blocks
- Perfect for documentation and sharing

## Usage Instructions

### VS Code Integration
- **HTML files**: Open directly in VS Code for best experience
- **ANSI files**: Use \`cat filename.ansi\` in VS Code terminal
- **Markdown files**: Use VS Code's built-in markdown preview
- **Text files**: Standard text file support

### Terminal Viewing
- **ANSI files**: \`cat filename.ansi\` or \`less -R filename.ansi\`
- **Text files**: \`cat filename.txt\` or \`less filename.txt\`
- **Markdown files**: Use any markdown viewer or \`cat filename.md\`

---

Generated by SSHXtend - Enhanced collaborative terminal
https://github.com/ovidiuvio/sshx
`;
  }

  /**
   * Download the export result as a file
   */
  downloadExport(result: ExportResult): void {
    try {
      let blob: Blob;
      
      if (result.mimeType === 'application/zip') {
        // Handle zip files specially (content is already Uint8Array)
        blob = new Blob([result.content], { type: result.mimeType });
      } else {
        // Handle text-based files
        blob = new Blob([result.content], { type: result.mimeType });
      }
      
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      
      a.href = url;
      a.download = result.filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Export download failed:', error);
      throw new Error(`Failed to download export: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Get available export formats with descriptions
   */
  getAvailableFormats(): Array<{format: ExportFormat, name: string, description: string, recommended?: boolean}> {
    return [
      {
        format: 'html',
        name: '🌐 HTML',
        description: 'Perfect for VS Code with preserved colors and formatting',
        recommended: true
      },
      {
        format: 'ansi',
        name: '🎨 ANSI',
        description: 'Raw terminal codes - view with `cat` in VS Code terminal'
      },
      {
        format: 'markdown',
        name: '📝 Markdown',
        description: 'Documentation-friendly with code blocks'
      },
      {
        format: 'txt',
        name: '📄 Plain Text',
        description: 'Simple text format (current default)'
      },
      {
        format: 'zip',
        name: '🗜️ All Formats (ZIP)',
        description: 'All formats in one zip archive'
      }
    ];
  }

  /**
   * Update theme and terminal info (for dynamic updates)
   */
  updateContext(theme: ExportTheme, terminalInfo: TerminalInfo): void {
    this.theme = theme;
    this.terminalInfo = terminalInfo;
    
    // Recreate exporters with new context
    this.htmlExporter = new HTMLExporter(this.term, theme, terminalInfo);
    this.ansiExporter = new ANSIExporter(this.term, terminalInfo);
    this.markdownExporter = new MarkdownExporter(this.term, terminalInfo);
  }

  private extractPlainTextContent(): string {
    // Replicate the existing downloadTerminalText logic
    try {
      // Try selection method first
      this.term.selectAll();
      const selectedContent = this.term.getSelection();
      this.term.clearSelection();

      if (selectedContent && selectedContent.trim()) {
        return selectedContent;
      }
    } catch (e) {
      console.warn("Selection method failed:", e);
    }

    // Fallback to buffer method
    try {
      const buffer = this.term.buffer.active;
      const lines: string[] = [];

      for (let i = 0; i < buffer.length; i++) {
        const line = buffer.getLine(i);
        if (line) {
          const lineText = line.translateToString(true);
          lines.push(lineText);
        }
      }

      const content = lines.join('\n');
      return content.trim() || 'No terminal content available';
    } catch (e) {
      console.error("Buffer method failed:", e);
      const errorMessage = e instanceof Error ? e.message : String(e);
      return `Error extracting terminal content: ${errorMessage}`;
    }
  }

  private generatePlainTextFilename(options: ExportOptions): string {
    const timestamp = new Date().toISOString().slice(0, 19).replace(/:/g, "-");
    const title = (options.title || this.terminalInfo.title || 'terminal')
      .replace(/[^a-zA-Z0-9]/g, "_")
      .toLowerCase();
    
    const suffix = options.selectionOnly ? 'selection' : 'session';
    return `terminal-${title}-${suffix}-${timestamp}.txt`;
  }
}