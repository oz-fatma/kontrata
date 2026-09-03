import {
  app,
  BrowserWindow,
  dialog,
  ipcMain,
  net,
  protocol,
} from "electron";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { startBackend, stopBackend, isBackendRunning } from "./backend";
import { loadSettings, saveSettings } from "./config";
import { apiBase, isDev, setupHtml, webRoot } from "./paths";
import { readRefreshToken, writeRefreshToken } from "./tokens";

protocol.registerSchemesAsPrivileged([
  {
    scheme: "kontrata",
    privileges: {
      standard: true,
      secure: true,
      supportFetchAPI: true,
      stream: true,
    },
  },
]);

let mainWindow: BrowserWindow | null = null;
let setupWindow: BrowserWindow | null = null;
let quitting = false;

const preloadPath = path.join(__dirname, "preload.js");

function windowPrefs(): Electron.BrowserWindowConstructorOptions {
  return {
    width: 1200,
    height: 800,
    minWidth: 880,
    minHeight: 600,
    show: false,
    autoHideMenuBar: true,
    webPreferences: {
      preload: preloadPath,
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      additionalArguments: [`--kontrata-api=${apiBase()}`],
    },
  };
}

function registerUiProtocol(): void {
  const root = path.resolve(webRoot());
  protocol.handle("kontrata", async (request) => {
    const u = new URL(request.url);
    let pathname = decodeURIComponent(u.pathname);
    if (pathname === "/" || pathname === "") {
      pathname = "/index.html";
    } else if (pathname.endsWith("/")) {
      pathname = `${pathname}index.html`;
    }
    const file = path.resolve(path.join(root, pathname));
    const prefix = root.endsWith(path.sep) ? root : root + path.sep;
    if (file !== root && !file.startsWith(prefix)) {
      return new Response("not found", { status: 404 });
    }
    try {
      return await net.fetch(pathToFileURL(file).href);
    } catch {
      return new Response("not found", { status: 404 });
    }
  });
}

function createMainWindow(): BrowserWindow {
  const win = new BrowserWindow(windowPrefs());
  win.once("ready-to-show", () => win.show());
  if (isDev()) {
    void win.loadURL("http://localhost:3000");
  } else {
    void win.loadURL("kontrata://app/");
  }
  win.on("closed", () => {
    if (mainWindow === win) {
      mainWindow = null;
    }
  });
  return win;
}

function createSetupWindow(): BrowserWindow {
  const win = new BrowserWindow({
    ...windowPrefs(),
    width: 520,
    height: 640,
  });
  win.once("ready-to-show", () => win.show());
  void win.loadFile(setupHtml());
  win.on("closed", () => {
    if (setupWindow === win) {
      setupWindow = null;
    }
  });
  return win;
}

async function waitForDevUi(): Promise<void> {
  if (!isDev()) {
    return;
  }
  const deadline = Date.now() + 30_000;
  while (Date.now() < deadline) {
    try {
      const res = await fetch("http://localhost:3000");
      if (res.ok || res.status === 404) {
        return;
      }
    } catch {
      /* Next henüz ayakta değil */
    }
    await new Promise((r) => setTimeout(r, 250));
  }
  throw new Error("arayüz geliştirme sunucusuna ulaşılamadı (http://localhost:3000)");
}

async function openApp(): Promise<void> {
  const settings = loadSettings();
  if (!settings) {
    setupWindow = createSetupWindow();
    return;
  }
  try {
    await startBackend(settings);
    await waitForDevUi();
  } catch (err) {
    const msg = err instanceof Error ? err.message : "arka plan servisi başlatılamadı";
    dialog.showErrorBox("Kontrata", msg);
    setupWindow = createSetupWindow();
    return;
  }
  mainWindow = createMainWindow();
}

function bindIpc(): void {
  ipcMain.handle("token:get", () => readRefreshToken());
  ipcMain.handle("token:set", (_e, token: unknown) => {
    if (token !== null && typeof token !== "string") {
      return;
    }
    writeRefreshToken(token);
  });
  ipcMain.handle(
    "setup:save",
    async (
      _e,
      input: { mongoUri?: string; llmEndpointUrl?: string; llmToken?: string },
    ): Promise<{ ok: true } | { ok: false; error: string }> => {
      const mongoUri = typeof input?.mongoUri === "string" ? input.mongoUri.trim() : "";
      if (!mongoUri) {
        return { ok: false, error: "MONGO_URI gerekli" };
      }
      try {
        const settings = saveSettings({
          mongoUri,
          llmEndpointUrl: typeof input.llmEndpointUrl === "string" ? input.llmEndpointUrl : "",
          llmToken: typeof input.llmToken === "string" ? input.llmToken : "",
        });
        await startBackend(settings);
        await waitForDevUi();
      } catch (err) {
        const error = err instanceof Error ? err.message : "kayıt başarısız";
        return { ok: false, error };
      }
      if (!mainWindow) {
        mainWindow = createMainWindow();
      }
      setupWindow?.close();
      setupWindow = null;
      return { ok: true };
    },
  );
}

const gotLock = app.requestSingleInstanceLock();
if (!gotLock) {
  app.quit();
} else {
  app.on("second-instance", () => {
    const win = mainWindow ?? setupWindow;
    if (win) {
      if (win.isMinimized()) {
        win.restore();
      }
      win.focus();
    }
  });

  app.whenReady().then(() => {
    bindIpc();
    if (!isDev()) {
      registerUiProtocol();
    }
    void openApp();
  });

  app.on("activate", () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      if (isBackendRunning()) {
        mainWindow = createMainWindow();
        return;
      }
      void openApp();
    }
  });

  app.on("window-all-closed", () => {
    if (process.platform !== "darwin") {
      app.quit();
    }
  });

  app.on("before-quit", (e) => {
    if (quitting) {
      return;
    }
    e.preventDefault();
    quitting = true;
    void stopBackend().finally(() => {
      app.quit();
    });
  });
}
