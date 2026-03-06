import { ipcRenderer } from 'electron';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { SerializeAddon } from '@xterm/addon-serialize';
import { WebglAddon } from '@xterm/addon-webgl';

import '@xterm/xterm/css/xterm.css';
import { HTMLExporter } from './HTMLExporter';

// Initialize terminal
const term = new Terminal({
  cursorBlink: true,
  fontFamily: 'Consolas, "Courier New", monospace',
  fontSize: 14,
  theme: {
    background: '#1e1e1e',
    foreground: '#d4d4d4',
  },
  scrollback: 1000000 // practically infinite scrollback
});

const fitAddon = new FitAddon();
const serializeAddon = new SerializeAddon();

term.loadAddon(fitAddon);
term.loadAddon(serializeAddon);

const container = document.getElementById('terminal-container');
if (container) {
  term.open(container);
  try {
      const webgl = new WebglAddon();
      term.loadAddon(webgl);
  } catch (e) {
      console.warn("WebGL addon could not load, falling back to canvas/dom renderer.");
  }
  fitAddon.fit();
}

// IPC Setup
ipcRenderer.on('terminal.incData', (event, data) => {
  term.write(data);
});

term.onData((data) => {
  ipcRenderer.send('terminal.toTerm', data);
});

// Resize handling
window.addEventListener('resize', () => {
  fitAddon.fit();
});

term.onResize(({ cols, rows }) => {
  ipcRenderer.send('terminal.resize', { cols, rows });
});

// Initial resize
setTimeout(() => {
    fitAddon.fit();
    ipcRenderer.send('terminal.resize', { cols: term.cols, rows: term.rows });
}, 100);

// Export Handling
const btnHtml = document.getElementById('btn-export-html');
const btnAnsi = document.getElementById('btn-export-ansi');
const btnTxt = document.getElementById('btn-export-txt');

if (btnHtml) {
  btnHtml.addEventListener('click', async () => {
    const exporter = new HTMLExporter(term, serializeAddon, {
      background: '#1e1e1e',
      foreground: '#d4d4d4'
    }, {
      fontFamily: 'Consolas, "Courier New", monospace',
      fontSize: 14,
      title: 'Local Terminal'
    });

    const htmlContent = exporter.exportHTML();

    const timestamp = new Date().toISOString().slice(0, 19).replace(/:/g, "-");
    const result = await ipcRenderer.invoke('save-file', {
      title: 'Export Terminal to HTML',
      defaultPath: `terminal-session-${timestamp}.html`,
      content: htmlContent,
      filters: [{ name: 'HTML Files', extensions: ['html'] }]
    });

    if (result && !result.canceled && result.filePath) {
      console.log('Saved to', result.filePath);
    }
  });
}

if (btnAnsi) {
  btnAnsi.addEventListener('click', async () => {
     // fallback to ANSI export using serialize addon without HTML formatting
    const ansiContent = serializeAddon.serialize();
    const timestamp = new Date().toISOString().slice(0, 19).replace(/:/g, "-");
    await ipcRenderer.invoke('save-file', {
      title: 'Export Terminal to ANSI',
      defaultPath: `terminal-session-${timestamp}.ansi`,
      content: ansiContent,
      filters: [{ name: 'ANSI Files', extensions: ['ansi', 'txt'] }]
    });
  });
}

if (btnTxt) {
    btnTxt.addEventListener('click', async () => {
        let content = "";
        const buffer = term.buffer.active;
        for (let i = 0; i < buffer.length; i++) {
            const line = buffer.getLine(i);
            if (line) {
                content += line.translateToString(true) + "\n";
            }
        }

        const timestamp = new Date().toISOString().slice(0, 19).replace(/:/g, "-");
        await ipcRenderer.invoke('save-file', {
        title: 'Export Terminal to Text',
        defaultPath: `terminal-session-${timestamp}.txt`,
        content: content,
        filters: [{ name: 'Text Files', extensions: ['txt'] }]
        });
    });
}
console.log("Renderer initialized.");
