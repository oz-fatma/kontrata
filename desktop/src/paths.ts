import { app } from "electron";
import fs from "node:fs";
import path from "node:path";

export const BACKEND_PORT = 17890;

export function isDev(): boolean {
  return process.env.NODE_ENV === "development" && !app.isPackaged;
}

export function backendBinary(): string {
  const name = process.platform === "win32" ? "api.exe" : "api";
  if (app.isPackaged) {
    return path.join(process.resourcesPath, name);
  }
  const local = path.join(__dirname, "..", "resources", "bin", name);
  if (fs.existsSync(local)) {
    return local;
  }
  const goBin =
    process.platform === "win32"
      ? path.join(__dirname, "..", "..", "backend", "bin", "api.exe")
      : path.join(__dirname, "..", "..", "backend", "bin", "api");
  return goBin;
}

export function webRoot(): string {
  if (app.isPackaged) {
    return path.join(process.resourcesPath, "web");
  }
  return path.join(__dirname, "..", "resources", "web");
}

export function setupHtml(): string {
  return path.join(__dirname, "setup.html");
}

export function apiBase(): string {
  return `http://127.0.0.1:${BACKEND_PORT}`;
}

export function userDataPath(...parts: string[]): string {
  return path.join(app.getPath("userData"), ...parts);
}
