/**
 * Simple markdown to ANSI converter for browser use with xterm.js
 * This is a lightweight alternative to marked-terminal which doesn't work in browsers
 */

// ANSI escape codes
const ANSI = {
  reset: '\x1b[0m',
  bold: '\x1b[1m',
  dim: '\x1b[2m',
  italic: '\x1b[3m',
  underline: '\x1b[4m',
  
  // Colors
  black: '\x1b[30m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m',
  white: '\x1b[37m',
  gray: '\x1b[90m',
  
  // Bright colors
  brightRed: '\x1b[91m',
  brightGreen: '\x1b[92m',
  brightYellow: '\x1b[93m',
  brightBlue: '\x1b[94m',
  brightMagenta: '\x1b[95m',
  brightCyan: '\x1b[96m',
};

function processMarkdownTables(text: string): string {
  // Match markdown tables
  const tableRegex = /(\|[^\n]+\|\n)(\|[\s:|-]+\|\n)((?:\|[^\n]+\|\n?)+)/gm;
  
  return text.replace(tableRegex, (match, headerLine, separatorLine, bodyLines) => {
    // Parse header
    const headers = headerLine.split('|')
      .filter((h: string) => h.trim())
      .map((h: string) => h.trim());
    
    // Parse alignment from separator
    const alignments = separatorLine.split('|')
      .filter((a: string) => a.trim())
      .map((a: string) => {
        const trimmed = a.trim();
        if (trimmed.startsWith(':') && trimmed.endsWith(':')) return 'center';
        if (trimmed.endsWith(':')) return 'right';
        return 'left';
      });
    
    // Parse body rows
    const rows = bodyLines.trim().split('\n').map((line: string) =>
      line.split('|')
        .filter((cell: string) => cell !== '')
        .map((cell: string) => cell.trim())
    );
    
    // Calculate column widths
    const colWidths = headers.map((h: string, i: number) => {
      const maxWidth = Math.max(
        h.length,
        ...rows.map((row: string[]) => (row[i] || '').length)
      );
      // Limit column width for terminal display
      // Adjust based on number of columns to fit better
      const maxColWidth = headers.length > 4 ? 25 : headers.length > 3 ? 30 : 40;
      return Math.min(maxWidth, maxColWidth);
    });
    
    // Format the table
    let result = '';
    
    // Top border
    result += `${ANSI.gray}┌`;
    result += colWidths.map((w: number) => '─'.repeat(w + 2)).join('┬');
    result += `┐${ANSI.reset}\n`;
    
    // Header row
    result += `${ANSI.gray}│${ANSI.reset} `;
    result += headers.map((h: string, i: number) => {
      const content = h.length > colWidths[i] 
        ? h.substring(0, colWidths[i] - 3) + '...'
        : h;
      return `${ANSI.bold}${ANSI.cyan}${content.padEnd(colWidths[i])}${ANSI.reset}`;
    }).join(` ${ANSI.gray}│${ANSI.reset} `);
    result += ` ${ANSI.gray}│${ANSI.reset}\n`;
    
    // Header separator
    result += `${ANSI.gray}├`;
    result += colWidths.map((w: number) => '─'.repeat(w + 2)).join('┼');
    result += `┤${ANSI.reset}\n`;
    
    // Body rows
    for (const row of rows) {
      result += `${ANSI.gray}│${ANSI.reset} `;
      result += row.map((cell: string, i: number) => {
        const content = cell.length > colWidths[i] 
          ? cell.substring(0, colWidths[i] - 3) + '...'
          : cell;
        
        // Apply alignment
        let aligned;
        if (alignments[i] === 'right') {
          aligned = content.padStart(colWidths[i]);
        } else if (alignments[i] === 'center') {
          const padding = colWidths[i] - content.length;
          const leftPad = Math.floor(padding / 2);
          const rightPad = padding - leftPad;
          aligned = ' '.repeat(leftPad) + content + ' '.repeat(rightPad);
        } else {
          aligned = content.padEnd(colWidths[i]);
        }
        
        return aligned;
      }).join(` ${ANSI.gray}│${ANSI.reset} `);
      result += ` ${ANSI.gray}│${ANSI.reset}\n`;
    }
    
    // Bottom border
    result += `${ANSI.gray}└`;
    result += colWidths.map((w: number) => '─'.repeat(w + 2)).join('┴');
    result += `┘${ANSI.reset}`;
    
    return result;
  });
}

export function markdownToAnsi(markdown: string): string {
  let result = markdown;
  
  // Process tables first (before other replacements)
  result = processMarkdownTables(result);
  
  // Headers (# to ######)
  result = result.replace(/^(#{1,6})\s+(.+)$/gm, (match, hashes, content) => {
    const level = hashes.length;
    if (level === 1) {
      return `${ANSI.bold}${ANSI.brightMagenta}${content}${ANSI.reset}`;
    } else if (level === 2) {
      return `${ANSI.bold}${ANSI.brightCyan}${content}${ANSI.reset}`;
    } else {
      return `${ANSI.bold}${ANSI.cyan}${content}${ANSI.reset}`;
    }
  });
  
  // Bold text **text** or __text__
  result = result.replace(/\*\*(.+?)\*\*|__(.+?)__/g, (match, p1, p2) => {
    const content = p1 || p2;
    return `${ANSI.bold}${content}${ANSI.reset}`;
  });
  
  // Italic text *text* or _text_ (but not ** or __)
  result = result.replace(/(?<!\*)\*(?!\*)(.+?)(?<!\*)\*(?!\*)|(?<!_)_(?!_)(.+?)(?<!_)_(?!_)/g, (match, p1, p2) => {
    const content = p1 || p2;
    return `${ANSI.italic}${content}${ANSI.reset}`;
  });
  
  // Inline code `code`
  result = result.replace(/`([^`]+)`/g, (match, code) => {
    return `${ANSI.brightYellow}${code}${ANSI.reset}`;
  });
  
  // Code blocks with language ```lang\ncode``` or ```code```
  result = result.replace(/```(\w*)\n?([\s\S]*?)```/gm, (match, lang, code) => {
    // Handle empty code blocks
    if (!code || !code.trim()) {
      return match;
    }
    
    // Remove leading/trailing whitespace but preserve internal structure
    const trimmedCode = code.replace(/^\n+|\n+$/g, '');
    const lines = trimmedCode.split('\n');
    
    // Calculate max line length for proper box drawing
    const maxLength = Math.min(80, Math.max(
      ...lines.map((l: string) => l.length),
      lang ? lang.length + 4 : 10
    ));
    
    // Format each line with proper padding
    const formattedLines = lines.map((line: string) => {
      // Truncate very long lines
      const displayLine = line.length > maxLength ? 
        line.substring(0, maxLength - 3) + '...' : 
        line.padEnd(maxLength);
      return `${ANSI.gray}│${ANSI.reset} ${ANSI.brightYellow}${displayLine}${ANSI.reset} ${ANSI.gray}│${ANSI.reset}`;
    });
    
    // Create header with language label if provided
    const headerWidth = maxLength + 2; // +2 for spaces on sides
    if (lang) {
      const langLabel = ` ${lang} `;
      const leftPadding = Math.floor((headerWidth - langLabel.length) / 2);
      const rightPadding = headerWidth - langLabel.length - leftPadding;
      const header = `${ANSI.gray}┌${'─'.repeat(leftPadding)}${ANSI.cyan}${langLabel}${ANSI.gray}${'─'.repeat(rightPadding)}┐${ANSI.reset}`;
      const footer = `${ANSI.gray}└${'─'.repeat(headerWidth)}┘${ANSI.reset}`;
      
      return `\n${header}\n${formattedLines.join('\n')}\n${footer}\n`;
    } else {
      const header = `${ANSI.gray}┌${'─'.repeat(headerWidth)}┐${ANSI.reset}`;
      const footer = `${ANSI.gray}└${'─'.repeat(headerWidth)}┘${ANSI.reset}`;
      
      return `\n${header}\n${formattedLines.join('\n')}\n${footer}\n`;
    }
  });
  
  // Blockquotes > text
  result = result.replace(/^>\s+(.+)$/gm, (match, content) => {
    return `${ANSI.gray}│ ${content}${ANSI.reset}`;
  });
  
  // Process lists line by line to handle nesting
  const lines = result.split('\n');
  const processedLines = [];
  let inListItem = false;
  let currentIndentLevel = 0;
  
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    
    // Check for unordered lists with nesting
    const unorderedMatch = line.match(/^(\s*)([\*\-\+])\s+(.+)$/);
    if (unorderedMatch) {
      const [, indent, marker, content] = unorderedMatch;
      const indentLevel = Math.floor(indent.length / 2);
      const padding = '  '.repeat(indentLevel);
      
      // Use different bullets for different levels
      const bullets = ['•', '◦', '▪', '▫'];
      const bullet = bullets[Math.min(indentLevel, bullets.length - 1)];
      
      processedLines.push(`${padding}${ANSI.yellow}${bullet}${ANSI.reset} ${content}`);
      inListItem = true;
      currentIndentLevel = indentLevel;
      continue;
    }
    
    // Check for ordered lists
    const orderedMatch = line.match(/^(\s*)(\d+)\.\s+(.+)$/);
    if (orderedMatch) {
      const [, indent, num, content] = orderedMatch;
      const indentLevel = Math.floor(indent.length / 2);
      const padding = '  '.repeat(indentLevel);
      processedLines.push(`${padding}${ANSI.yellow}${num}.${ANSI.reset} ${content}`);
      inListItem = true;
      currentIndentLevel = indentLevel;
      continue;
    }
    
    // Check for continuation of list item (indented content without marker)
    if (inListItem && line.match(/^\s+/) && !line.match(/^(\s*)([\*\-\+]|\d+\.)\s+/)) {
      const contentIndent = '  '.repeat(currentIndentLevel + 1);
      processedLines.push(`${contentIndent}${line.trim()}`);
      continue;
    }
    
    // If we hit a non-list line, reset list state
    if (!line.match(/^\s*$/) && !line.match(/^(\s*)([\*\-\+]|\d+\.)\s+/)) {
      inListItem = false;
      currentIndentLevel = 0;
    }
    
    processedLines.push(line);
  }
  
  result = processedLines.join('\n');
  
  // Horizontal rules (---, ***, ___)
  result = result.replace(/^(---|\*\*\*|___)$/gm, () => {
    return `${ANSI.gray}${'─'.repeat(40)}${ANSI.reset}`;
  });
  
  // Links [text](url)
  result = result.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (match, text, url) => {
    return `${ANSI.brightBlue}${text}${ANSI.reset} ${ANSI.gray}(${url})${ANSI.reset}`;
  });
  
  // Strikethrough ~~text~~
  result = result.replace(/~~(.+?)~~/g, (match, content) => {
    return `${ANSI.dim}${content}${ANSI.reset}`;
  });
  
  return result;
}