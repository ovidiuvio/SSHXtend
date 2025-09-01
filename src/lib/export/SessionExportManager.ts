// Session-wide export manager - coordinates exporting all terminals in a session
import JSZip from 'jszip';
import type { ExportFormat, ExportOptions, ExportResult } from './types';

export interface TerminalExportFunction {
  (options: ExportOptions): Promise<ExportResult> | ExportResult;
}

export interface SessionTerminalInfo {
  id: number;
  title?: string;
  exportFunction: TerminalExportFunction;
}

export class SessionExportManager {
  constructor(private terminals: SessionTerminalInfo[]) {}

  /**
   * Export all terminals in the session in the specified format
   */
  async exportSession(options: ExportOptions): Promise<ExportResult> {
    if (this.terminals.length === 0) {
      throw new Error('No terminals available for export');
    }

    switch (options.format) {
      case 'zip':
        return this.exportAllFormatsZip(options);
      default:
        return this.exportSingleFormatZip(options);
    }
  }

  /**
   * Export all terminals in a single format as a zip
   */
  private async exportSingleFormatZip(options: ExportOptions): Promise<ExportResult> {
    const zip = new JSZip();
    const timestamp = new Date().toISOString().slice(0, 19).replace(/:/g, "-");
    const formatName = this.getFormatDisplayName(options.format);
    
    // Create a folder for this format
    const formatFolder = zip.folder(formatName.toLowerCase());
    if (!formatFolder) {
      throw new Error(`Failed to create folder for format: ${formatName}`);
    }

    // Export each terminal in the specified format
    for (const terminal of this.terminals) {
      try {
        const terminalOptions = {
          ...options,
          title: terminal.title || `Terminal ${terminal.id}`
        };
        
        const result = await terminal.exportFunction(terminalOptions);
        // Ensure unique filename by prefixing with terminal ID
        const uniqueFilename = `terminal-${terminal.id}-${result.filename}`;
        formatFolder.file(uniqueFilename, result.content);
      } catch (error) {
        console.error(`Failed to export terminal ${terminal.id}:`, error);
        // Add error file instead
        const errorContent = `Error exporting terminal ${terminal.id}: ${error instanceof Error ? error.message : 'Unknown error'}`;
        formatFolder.file(`terminal-${terminal.id}-error.txt`, errorContent);
      }
    }

    // Add session info
    const sessionInfo = this.generateSessionInfo(options);
    formatFolder.file('session-info.txt', sessionInfo);

    // Generate zip
    const zipContent = await zip.generateAsync({ 
      type: 'uint8array',
      compression: 'DEFLATE',
      compressionOptions: { level: 6 }
    });

    const suffix = options.selectionOnly ? 'selections' : 'session';
    return {
      content: zipContent as any,
      filename: `session-${formatName.toLowerCase()}-${suffix}-${timestamp}.zip`,
      mimeType: 'application/zip'
    };
  }

  /**
   * Export all terminals in all formats as a comprehensive zip archive
   */
  private async exportAllFormatsZip(options: ExportOptions): Promise<ExportResult> {
    const zip = new JSZip();
    const timestamp = new Date().toISOString().slice(0, 19).replace(/:/g, "-");
    const formats: ExportFormat[] = ['html', 'ansi', 'markdown', 'txt'];
    
    // Create a folder for each format
    for (const format of formats) {
      const formatName = this.getFormatDisplayName(format);
      const formatFolder = zip.folder(formatName.toLowerCase());
      if (!formatFolder) continue;

      // Export each terminal in this format
      for (const terminal of this.terminals) {
        try {
          const terminalOptions = {
            ...options,
            format,
            title: terminal.title || `Terminal ${terminal.id}`
          };
          
          const result = await terminal.exportFunction(terminalOptions);
          // Ensure unique filename by prefixing with terminal ID
          const uniqueFilename = `terminal-${terminal.id}-${result.filename}`;
          formatFolder.file(uniqueFilename, result.content);
        } catch (error) {
          console.error(`Failed to export terminal ${terminal.id} in ${format} format:`, error);
          // Add error file instead
          const errorContent = `Error exporting terminal ${terminal.id} in ${format} format: ${error instanceof Error ? error.message : 'Unknown error'}`;
          formatFolder.file(`terminal-${terminal.id}-${format}-error.txt`, errorContent);
        }
      }
    }

    // Add comprehensive README
    const readme = this.generateComprehensiveReadme(options);
    zip.file('README.txt', readme);

    // Add session info
    const sessionInfo = this.generateSessionInfo(options);
    zip.file('session-info.txt', sessionInfo);

    // Generate zip
    const zipContent = await zip.generateAsync({ 
      type: 'uint8array',
      compression: 'DEFLATE',
      compressionOptions: { level: 6 }
    });

    const suffix = options.selectionOnly ? 'selections' : 'session';
    return {
      content: zipContent as any,
      filename: `session-complete-${suffix}-${timestamp}.zip`,
      mimeType: 'application/zip'
    };
  }

  /**
   * Generate session information file
   */
  private generateSessionInfo(options: ExportOptions): string {
    const timestamp = new Date().toISOString();
    const terminalCount = this.terminals.length;
    const terminalList = this.terminals
      .map(t => `  - Terminal ${t.id}: ${t.title || 'Untitled'}`)
      .join('\n');

    return `# Session Export Information

Exported: ${new Date(timestamp).toLocaleString()}
Export Type: ${options.selectionOnly ? 'Selected text only' : 'Full terminal sessions'}
Format: ${options.format === 'zip' ? 'All formats' : this.getFormatDisplayName(options.format)}
Terminal Count: ${terminalCount}

## Terminals Included

${terminalList}

## Export Options

- Selection Only: ${options.selectionOnly ? 'Yes' : 'No'}
- Include Timestamp: ${options.includeTimestamp !== false ? 'Yes' : 'No'}
- Optimize for VS Code: ${options.optimizeForVSCode !== false ? 'Yes' : 'No'}

---

Generated by SSHXtend - Enhanced collaborative terminal
https://github.com/ovidiuvio/sshx
`;
  }

  /**
   * Generate comprehensive README for all-formats export
   */
  private generateComprehensiveReadme(options: ExportOptions): string {
    const timestamp = new Date().toISOString();
    const terminalCount = this.terminals.length;
    
    return `# Session Export Archive - All Formats

Exported: ${new Date(timestamp).toLocaleString()}
Source: SSHXtend Terminal Export
Terminal Count: ${terminalCount}
${options.selectionOnly ? 'Content: Selected text only from all terminals' : 'Content: Complete terminal sessions'}

## Archive Structure

This archive contains exports of all terminals in your session across multiple formats:

### 📁 html/
- **HTML Format** (Recommended for VS Code)
- Perfect color and formatting preservation
- Opens directly in VS Code with full styling
- CSS optimized for VS Code display
- Mobile responsive design

### 📁 ansi/
- **ANSI Format** (Terminal Native)
- Raw terminal escape sequences preserved
- View with: \`cat filename.ansi\`
- Compatible with any terminal emulator
- Native terminal color support

### 📁 markdown/
- **Markdown Format** (Documentation)
- Documentation-friendly format
- GitHub-compatible rendering
- Syntax highlighting in code blocks
- Perfect for documentation and sharing

### 📁 txt/
- **Plain Text Format** (Universal)
- Simple text format without formatting
- Universal compatibility
- Smallest file size
- Works everywhere

## Usage Instructions

### VS Code Integration
- **HTML files**: Open directly in VS Code for best experience
- **ANSI files**: Use \`cat filename.ansi\` in VS Code terminal
- **Markdown files**: Use VS Code's built-in markdown preview
- **Text files**: Standard text file support

### Terminal Viewing
- **ANSI files**: \`cat filename.ansi\` or \`less -R filename.ansi\`
- **Text files**: \`cat filename.txt\` or \`less filename.txt\`
- **Markdown files**: Use any markdown viewer

### File Naming Convention
Files are named: \`terminal-{title}-{content-type}-{timestamp}.{extension}\`
- **title**: Terminal title or "terminal-{id}" if untitled
- **content-type**: "session" (full content) or "selection" (selected text only)
- **timestamp**: ISO format timestamp (YYYY-MM-DDTHH-MM-SS)

## Terminal List

${this.terminals.map(t => `- Terminal ${t.id}: ${t.title || 'Untitled'}`).join('\n')}

---

Generated by SSHXtend - Enhanced collaborative terminal
https://github.com/ovidiuvio/sshx

For more information about export formats and usage, see session-info.txt
`;
  }

  /**
   * Get display name for format
   */
  private getFormatDisplayName(format: ExportFormat): string {
    switch (format) {
      case 'html': return 'HTML';
      case 'ansi': return 'ANSI';
      case 'markdown': return 'Markdown';
      case 'txt': return 'Text';
      case 'zip': return 'All Formats';
      default: return String(format).toUpperCase();
    }
  }

  /**
   * Download the export result
   */
  downloadExport(result: ExportResult): void {
    try {
      const blob = new Blob([result.content], { type: result.mimeType });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      
      a.href = url;
      a.download = result.filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Session export download failed:', error);
      throw new Error(`Failed to download session export: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Get available export formats for session export
   */
  getAvailableFormats(): Array<{format: ExportFormat, name: string, description: string, recommended?: boolean}> {
    return [
      {
        format: 'zip',
        name: '🗜️ All Formats (ZIP)',
        description: `Export all ${this.terminals.length} terminals in all available formats`,
        recommended: true
      },
      {
        format: 'html',
        name: '🌐 HTML (ZIP)',
        description: `Export all ${this.terminals.length} terminals as HTML files`
      },
      {
        format: 'ansi',
        name: '🎨 ANSI (ZIP)',
        description: `Export all ${this.terminals.length} terminals as ANSI files`
      },
      {
        format: 'markdown',
        name: '📝 Markdown (ZIP)',
        description: `Export all ${this.terminals.length} terminals as Markdown files`
      },
      {
        format: 'txt',
        name: '📄 Text (ZIP)',
        description: `Export all ${this.terminals.length} terminals as text files`
      }
    ];
  }

  /**
   * Update terminals list (for dynamic updates)
   */
  updateTerminals(terminals: SessionTerminalInfo[]): void {
    this.terminals = terminals;
  }
}