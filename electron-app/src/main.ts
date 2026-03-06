import { app, BrowserWindow, ipcMain, dialog } from 'electron';
import * as path from 'path';
import * as os from 'os';
import * as pty from 'node-pty';
import * as fs from 'fs';

// Handle creating/removing shortcuts on Windows when installing/uninstalling.
if (require('electron-squirrel-startup')) {
  app.quit();
}

let mainWindow: BrowserWindow | null = null;
let ptyProcess: pty.IPty | null = null;
let logStream: fs.WriteStream | null = null;

const createWindow = () => {
  mainWindow = new BrowserWindow({
    width: 1000,
    height: 600,
    webPreferences: {
      nodeIntegration: true,
      contextIsolation: false, // For simplicity in this local app. In a real app, use preload scripts.
    },
  });

  mainWindow.loadFile(path.join(__dirname, 'index.html'));

  mainWindow.on('closed', () => {
    mainWindow = null;
  });
};

const setupPty = () => {
  const shell = process.env[os.platform() === 'win32' ? 'COMSPEC' : 'SHELL'] || (os.platform() === 'win32' ? 'cmd.exe' : 'bash');

  ptyProcess = pty.spawn(shell, [], {
    name: 'xterm-color',
    cols: 80,
    rows: 24,
    cwd: process.env.HOME || process.cwd(),
    env: process.env as Record<string, string>
  });

  const logDir = path.join(app.getPath('userData'), 'logs');
  if (!fs.existsSync(logDir)) {
    fs.mkdirSync(logDir, { recursive: true });
  }
  const logFile = path.join(logDir, `session-${Date.now()}.log`);
  logStream = fs.createWriteStream(logFile, { flags: 'a' });

  ptyProcess.onData((data) => {
    if (mainWindow) {
      mainWindow.webContents.send('terminal.incData', data);
    }
    if (logStream) {
      logStream.write(data);
    }
  });

  ipcMain.on('terminal.toTerm', (event, data) => {
    if (ptyProcess) {
      ptyProcess.write(data);
    }
  });

  ipcMain.on('terminal.resize', (event, { cols, rows }) => {
    if (ptyProcess) {
      try {
        ptyProcess.resize(cols, rows);
      } catch (e) {
        console.error("Resize error", e);
      }
    }
  });
};

app.on('ready', () => {
  createWindow();
  setupPty();
});

app.on('window-all-closed', () => {
  if (logStream) {
    logStream.end();
  }
  if (ptyProcess) {
    ptyProcess.kill();
  }
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('activate', () => {
  if (mainWindow === null) {
    createWindow();
  }
});

// IPC Handler for saving files
ipcMain.handle('save-file', async (event, options) => {
  if (!mainWindow) return { canceled: true };
  const result = await dialog.showSaveDialog(mainWindow, {
    title: options.title || 'Save File',
    defaultPath: options.defaultPath,
    filters: options.filters
  });

  if (!result.canceled && result.filePath) {
    try {
      fs.writeFileSync(result.filePath, options.content);
      return { canceled: false, filePath: result.filePath };
    } catch (err: any) {
      return { canceled: false, error: err.message };
    }
  }
  return { canceled: true };
});
