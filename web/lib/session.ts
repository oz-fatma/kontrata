const REFRESH_KEY = "kontrata.refresh";
const DEVICE_KEY = "kontrata.device";

let accessToken: string | null = null;
let pendingPassword: string | null = null;
let memoryRefresh: string | null = null;
let hydrated = false;
let hydratePromise: Promise<void> | null = null;
let persistChain: Promise<void> = Promise.resolve();

function electronApi(): Window["kontrata"] | undefined {
  if (typeof window === "undefined") {
    return undefined;
  }
  return window.kontrata;
}

export async function hydrateRefreshToken(): Promise<void> {
  if (hydrated) {
    return;
  }
  if (hydratePromise) {
    await hydratePromise;
    return;
  }
  hydratePromise = (async () => {
    const api = electronApi();
    if (api?.getRefreshToken) {
      memoryRefresh = await api.getRefreshToken();
    } else if (typeof window !== "undefined") {
      memoryRefresh = sessionStorage.getItem(REFRESH_KEY);
    }
    hydrated = true;
  })();
  await hydratePromise;
}

export function getAccessToken(): string | null {
  return accessToken;
}

export function setAccessToken(token: string | null): void {
  accessToken = token;
}

export function getRefreshToken(): string | null {
  if (typeof window === "undefined") {
    return null;
  }
  if (electronApi()?.getRefreshToken) {
    return memoryRefresh;
  }
  return sessionStorage.getItem(REFRESH_KEY);
}

export function setRefreshToken(token: string | null): void {
  if (typeof window === "undefined") {
    return;
  }
  memoryRefresh = token;
  const api = electronApi();
  if (api?.setRefreshToken) {
    persistChain = persistChain.then(() => api.setRefreshToken(token)).catch(() => {
      /* depo yazımı başarısız; bellek kopyası durur */
    });
    return;
  }
  if (token) {
    sessionStorage.setItem(REFRESH_KEY, token);
  } else {
    sessionStorage.removeItem(REFRESH_KEY);
  }
}

export function clearSession(): void {
  accessToken = null;
  pendingPassword = null;
  setRefreshToken(null);
}

export function setPendingPassword(password: string | null): void {
  pendingPassword = password;
}

export function getPendingPassword(): string | null {
  return pendingPassword;
}

export function getDeviceId(): string {
  if (typeof window === "undefined") {
    return "";
  }
  let id = localStorage.getItem(DEVICE_KEY);
  if (!id) {
    id = crypto.randomUUID();
    localStorage.setItem(DEVICE_KEY, id);
  }
  return id;
}

export function userIdFromAccessToken(token: string): string | null {
  const parts = token.split(".");
  if (parts.length < 2) {
    return null;
  }
  try {
    const json = atob(parts[1].replace(/-/g, "+").replace(/_/g, "/"));
    const payload = JSON.parse(json) as { kullaniciId?: string };
    return payload.kullaniciId ?? null;
  } catch {
    return null;
  }
}

export const MFA_TTL_SEC = 120;

const publicPrefixes = ["/kayit", "/giris", "/dogrula", "/sifre-sifirla"];

export function isPublicPath(path: string): boolean {
  const normalized = path.replace(/\/$/, "") || "/";
  return publicPrefixes.some((p) => normalized === p || normalized.startsWith(`${p}/`));
}
