import { safeStorage } from "electron";
import crypto from "node:crypto";
import fs from "node:fs";
import { userDataPath } from "./paths";

export type AppSettings = {
  mongoUri: string;
  llmEndpointUrl: string;
  llmToken: string;
  jwtSecret: string;
  smtpHost?: string;
  smtpPort?: number;
  smtpUser?: string;
  smtpPassword?: string;
  smtpFrom?: string;
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
      smtpHost: parsed.smtpHost,
      smtpPort: parsed.smtpPort,
      smtpUser: parsed.smtpUser,
      smtpPassword: parsed.smtpPassword,
      smtpFrom: parsed.smtpFrom,
    };
  } catch {
    return null;
  }
}

export function smtpConfigured(input: {
  smtpHost?: string;
  smtpFrom?: string;
}): boolean {
  return Boolean(input.smtpHost?.trim() && input.smtpFrom?.trim());
}

export function saveSettings(input: {
  mongoUri: string;
  llmEndpointUrl: string;
  llmToken: string;
  smtpHost?: string;
  smtpPort?: string;
  smtpUser?: string;
  smtpPassword?: string;
  smtpFrom?: string;
  jwtSecret?: string;
}): AppSettings {
  if (!safeStorage.isEncryptionAvailable()) {
    throw new Error("sistem anahtarlığı kullanılamıyor");
  }
  const existing = loadSettings();
  const smtpHost = input.smtpHost?.trim() ?? "";
  const smtpFrom = input.smtpFrom?.trim() ?? "";
  const smtpUser = input.smtpUser?.trim() ?? "";
  const next: AppSettings = {
    mongoUri: input.mongoUri.trim(),
    llmEndpointUrl: input.llmEndpointUrl.trim(),
    llmToken: input.llmToken.trim() || existing?.llmToken || "",
    jwtSecret: input.jwtSecret ?? existing?.jwtSecret ?? newJwtSecret(),
  };
  if (smtpConfigured({ smtpHost, smtpFrom })) {
    const portRaw = input.smtpPort?.trim() ?? "";
    let smtpPort = 587;
    if (portRaw) {
      const n = Number(portRaw);
      if (!Number.isInteger(n) || n < 1 || n > 65535) {
        throw new Error("SMTP_PORT geçersiz");
      }
      smtpPort = n;
    }
    next.smtpHost = smtpHost;
    next.smtpPort = smtpPort;
    next.smtpFrom = smtpFrom;
    if (smtpUser) {
      next.smtpUser = smtpUser;
    }
    const pass = input.smtpPassword?.trim() || existing?.smtpPassword || "";
    if (pass) {
      next.smtpPassword = pass;
    }
  }
  const buf = safeStorage.encryptString(JSON.stringify(next));
  fs.writeFileSync(settingsPath(), buf, { mode: 0o600 });
  return next;
}

export function newJwtSecret(): string {
  return crypto.randomBytes(32).toString("hex");
}
