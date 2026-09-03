import { app, dialog } from "electron";
import { spawn, type ChildProcess } from "node:child_process";
import fs from "node:fs";
import { BACKEND_PORT, backendBinary, isDev, userDataPath } from "./paths";
import type { AppSettings } from "./config";

const healthTimeoutMs = 30_000;
const killWaitMs = 5_000;
const pollMs = 250;

let child: ChildProcess | null = null;
let shuttingDown = false;
let healthy = false;

export function backendEnv(settings: AppSettings): NodeJS.ProcessEnv {
  return {
    ...process.env,
    PORT: String(BACKEND_PORT),
    MONGO_URI: settings.mongoUri,
    MONGO_DATABASE: process.env.MONGO_DATABASE || "kontrata",
    JWT_SECRET: settings.jwtSecret,
    LLM_ENDPOINT_URL: settings.llmEndpointUrl,
    LLM_TOKEN: settings.llmToken,
    UPLOAD_DIR: userDataPath("uploads"),
    GRAPHQL_PLAYGROUND: "false",
    MAILER: "console",
  };
}

export function isBackendRunning(): boolean {
  return child != null && child.exitCode == null && healthy;
}

export async function startBackend(settings: AppSettings): Promise<void> {
  const bin = backendBinary();
  if (!fs.existsSync(bin)) {
    throw new Error("arka plan servisi bulunamadı; önce backend ikilisini derleyin");
  }
  await stopBackend();
  shuttingDown = false;
  healthy = false;
  fs.mkdirSync(userDataPath("uploads"), { recursive: true });
  const forwardLogs = isDev();
  child = spawn(bin, [], {
    env: backendEnv(settings),
    cwd: app.getPath("userData"),
    stdio: ["ignore", forwardLogs ? "pipe" : "ignore", forwardLogs ? "pipe" : "ignore"],
    windowsHide: true,
  });
  if (forwardLogs) {
    prefixChildOutput(child.stdout, process.stdout);
    prefixChildOutput(child.stderr, process.stderr);
  }
  child.on("exit", () => {
    child = null;
    const crashedAfterReady = healthy && !shuttingDown;
    healthy = false;
    if (!crashedAfterReady) {
      return;
    }
    dialog
      .showMessageBox({
        type: "error",
        title: "Kontrata",
        message: "Arka plan servisi durdu. Uygulama kapanacak.",
      })
      .finally(() => {
        app.quit();
      });
  });
  try {
    await waitForHealth();
    healthy = true;
  } catch (err) {
    shuttingDown = true;
    await stopBackend();
    throw err;
  }
}

export async function waitForHealth(): Promise<void> {
  const url = `http://127.0.0.1:${BACKEND_PORT}/healthz`;
  const deadline = Date.now() + healthTimeoutMs;
  let lastErr = "arka plan servisi 30 saniye içinde hazır olmadı";
  while (Date.now() < deadline) {
    if (child && child.exitCode != null) {
      throw new Error("arka plan servisi hemen kapandı");
    }
    try {
      const res = await fetch(url);
      if (res.ok) {
        return;
      }
      lastErr = res.status === 503 ? "MongoDB'ye bağlanılamadı" : `sağlık ${res.status}`;
    } catch {
      lastErr = "servis henüz yanıt vermiyor";
    }
    await sleep(pollMs);
  }
  throw new Error(lastErr);
}

export async function stopBackend(): Promise<void> {
  shuttingDown = true;
  healthy = false;
  const proc = child;
  child = null;
  if (!proc || proc.exitCode != null) {
    return;
  }
  await new Promise<void>((resolve) => {
    const done = (): void => resolve();
    proc.once("exit", done);
    if (process.platform === "win32") {
      proc.kill();
    } else {
      proc.kill("SIGTERM");
      const t = setTimeout(() => {
        if (proc.exitCode == null) {
          proc.kill("SIGKILL");
        }
      }, killWaitMs);
      proc.once("exit", () => clearTimeout(t));
    }
  });
}

function prefixChildOutput(
  stream: NodeJS.ReadableStream | null | undefined,
  out: NodeJS.WriteStream,
): void {
  if (!stream) {
    return;
  }
  let buf = "";
  stream.setEncoding("utf8");
  stream.on("data", (chunk: string) => {
    buf += chunk;
    const lines = buf.split(/\r?\n/);
    buf = lines.pop() ?? "";
    for (const line of lines) {
      out.write(`[backend] ${line}\n`);
    }
  });
  stream.on("end", () => {
    if (buf.length > 0) {
      out.write(`[backend] ${buf}\n`);
    }
  });
}

function sleep(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}
