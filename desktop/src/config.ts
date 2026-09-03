import { safeStorage } from "electron";
import crypto from "node:crypto";
import fs from "node:fs";
import { userDataPath } from "./paths";

export type AppSettings = {
  mongoUri: string;
  llmEndpointUrl: string;
  llmToken: string;
  jwtSecret: string;
};

const fileName = "settings.bin";

export function settingsPath(): string {
  return userDataPath(fileName);
}

export function hasSettings(): boolean {
  return fs.existsSync(settingsPath());
}

export function loadSettings(): AppSettings | null {
  const p = settingsPath();
  if (!fs.existsSync(p)) {
    return null;
  }
  if (!safeStorage.isEncryptionAvailable()) {
    return null;
  }
  try {
    const json = safeStorage.decryptString(fs.readFileSync(p));
    const parsed = JSON.parse(json) as Partial<AppSettings>;
    if (!parsed.mongoUri || !parsed.jwtSecret) {
      return null;
    }
    return {
      mongoUri: parsed.mongoUri,
      llmEndpointUrl: parsed.llmEndpointUrl ?? "",
      llmToken: parsed.llmToken ?? "",
      jwtSecret: parsed.jwtSecret,
    };
  } catch {
    return null;
  }
}

export function saveSettings(input: {
  mongoUri: string;
  llmEndpointUrl: string;
  llmToken: string;
  jwtSecret?: string;
}): AppSettings {
  if (!safeStorage.isEncryptionAvailable()) {
    throw new Error("sistem anahtarlığı kullanılamıyor");
  }
  const existing = loadSettings();
  const next: AppSettings = {
    mongoUri: input.mongoUri.trim(),
    llmEndpointUrl: input.llmEndpointUrl.trim(),
    llmToken: input.llmToken,
    jwtSecret: input.jwtSecret ?? existing?.jwtSecret ?? newJwtSecret(),
  };
  const buf = safeStorage.encryptString(JSON.stringify(next));
  fs.writeFileSync(settingsPath(), buf, { mode: 0o600 });
  return next;
}

export function newJwtSecret(): string {
  return crypto.randomBytes(32).toString("hex");
}
