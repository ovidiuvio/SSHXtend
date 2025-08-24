// Export system types and interfaces
export type ExportFormat = 'html' | 'ansi' | 'markdown' | 'txt' | 'zip';

export interface ExportOptions {
  format: ExportFormat;
  selectionOnly?: boolean;
  includeTimestamp?: boolean;
  optimizeForVSCode?: boolean;
  title?: string;
}

export interface ExportResult {
  content: string;
  filename: string;
  mimeType: string;
}

export interface ExportTheme {
  background: string;
  foreground: string;
  cursor?: string;
  selection?: string;
  [key: string]: string | undefined;
}

export interface TerminalInfo {
  title: string;
  rows: number;
  cols: number;
  fontFamily: string;
  fontSize: number;
}