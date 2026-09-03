import { safeStorage } from "electron";
import fs from "node:fs";
import { userDataPath } from "./paths";

const fileName = "refresh.bin";

function tokenPath(): string {
  return userDataPath(fileName);
}

export function readRefreshToken(): string | null {
  const p = tokenPath();
  if (!fs.existsSync(p)) {
    return null;
  }
  if (!safeStorage.isEncryptionAvailable()) {
    return null;
  }
  try {
    return safeStorage.decryptString(fs.readFileSync(p));
  } catch {
    return null;
  }
}

export function writeRefreshToken(token: string | null): void {
  const p = tokenPath();
  if (!token) {
    if (fs.existsSync(p)) {
      fs.unlinkSync(p);
    }
    return;
  }
  if (!safeStorage.isEncryptionAvailable()) {
    throw new Error("sistem anahtarlığı kullanılamıyor");
  }
  fs.writeFileSync(p, safeStorage.encryptString(token), { mode: 0o600 });
}
