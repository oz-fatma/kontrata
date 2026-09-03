import { ClientError, GraphQLClient } from "graphql-request";
import type { TypedDocumentNode } from "@graphql-typed-document-node/core";
import { print } from "graphql";
import { JetonYenileDocument } from "@/generated/graphql";
import {
  clearSession,
  getAccessToken,
  getDeviceId,
  getRefreshToken,
  isPublicPath,
  setAccessToken,
  setRefreshToken,
} from "./session";

const endpoint =
  process.env.NEXT_PUBLIC_GRAPHQL_URL ?? "http://localhost:8080/graphql";

export function apiRoot(): string {
  return endpoint.replace(/\/graphql\/?$/i, "");
}

const client = new GraphQLClient(endpoint, {
  fetch: (url, init) => fetch(url, { ...init, cache: "no-store" }),
  headers: () => {
    const headers: Record<string, string> = {
      "Accept-Language": "tr",
      "X-Device-Id": getDeviceId(),
    };
    const token = getAccessToken();
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
    return headers;
  },
});

export class AuthExpiredError extends Error {
  constructor() {
    super("oturum sonlandı");
    this.name = "AuthExpiredError";
  }
}

function isAuthFailure(err: unknown): boolean {
  if (!(err instanceof ClientError)) {
    return false;
  }
  if (err.response.status === 401) {
    return true;
  }
  const msgs = err.response.errors?.map((e) => e.message) ?? [];
  return msgs.some((m) => m.includes("kimlik doğrulaması gerekli"));
}

export function isForbidden(err: unknown): boolean {
  if (!(err instanceof ClientError)) {
    return false;
  }
  const msgs = err.response.errors?.map((e) => e.message) ?? [];
  return msgs.some((m) => m.includes("yetkiniz yok"));
}

function redirectToLogin(): void {
  if (typeof window === "undefined") {
    return;
  }
  if (isPublicPath(window.location.pathname)) {
    return;
  }
  window.location.assign("/giris/");
}

export function graphqlMessage(err: unknown): string {
  if (err instanceof ClientError) {
    const msg = err.response.errors?.[0]?.message;
    if (msg) {
      return msg;
    }
    if (err.response.status) {
      return "Sunucuya ulaşılamadı";
    }
  }
  if (err instanceof Error && err.message) {
    if (/fetch|network|Failed/i.test(err.message)) {
      return "Sunucuya ulaşılamadı";
    }
    return err.message;
  }
  return "İşlem tamamlanamadı";
}

let refreshInFlight: Promise<boolean> | null = null;

async function tryRefresh(): Promise<boolean> {
  if (refreshInFlight) {
    return refreshInFlight;
  }
  refreshInFlight = (async () => {
    const refresh = getRefreshToken();
    if (!refresh) {
      return false;
    }
    const savedAccess = getAccessToken();
    setAccessToken(null);
    try {
      const data = await client.request({
        document: JetonYenileDocument,
        variables: { yenilemeJetonu: refresh },
      });
      setAccessToken(data.jetonYenile.erisimJetonu);
      setRefreshToken(data.jetonYenile.yenilemeJetonu);
      return true;
    } catch {
      setAccessToken(savedAccess);
      return false;
    }
  })();
  try {
    return await refreshInFlight;
  } finally {
    refreshInFlight = null;
  }
}

export async function gqlRequest<TResult, TVars extends object = object>(
  document: TypedDocumentNode<TResult, TVars>,
  variables?: TVars,
): Promise<TResult> {
  try {
    return await requestOnce(document, variables);
  } catch (err) {
    if (!isAuthFailure(err)) {
      throw err;
    }
    const ok = await tryRefresh();
    if (!ok) {
      clearSession();
      redirectToLogin();
      throw new AuthExpiredError();
    }
    return await requestOnce(document, variables);
  }
}

async function requestOnce<TResult, TVars extends object>(
  document: TypedDocumentNode<TResult, TVars>,
  variables?: TVars,
): Promise<TResult> {
  if (containsFile(variables)) {
    return multipartRequest(document, variables ?? ({} as TVars));
  }
  return client.request<TResult>(document as never, variables as never);
}

function containsFile(value: unknown): boolean {
  if (typeof File !== "undefined" && value instanceof File) {
    return true;
  }
  if (Array.isArray(value)) {
    return value.some(containsFile);
  }
  if (value && typeof value === "object") {
    return Object.values(value).some(containsFile);
  }
  return false;
}

async function multipartRequest<TResult, TVars extends object>(
  document: TypedDocumentNode<TResult, TVars>,
  variables: TVars,
): Promise<TResult> {
  const query = print(document);
  const opsVars: Record<string, unknown> = { ...(variables as Record<string, unknown>) };
  const form = new FormData();
  let fileIndex = 0;
  const map: Record<string, string[]> = {};
  for (const [key, val] of Object.entries(opsVars)) {
    if (typeof File !== "undefined" && val instanceof File) {
      const idx = String(fileIndex);
      map[idx] = [`variables.${key}`];
      opsVars[key] = null;
      fileIndex += 1;
    }
  }
  form.append("operations", JSON.stringify({ query, variables: opsVars }));
  form.append("map", JSON.stringify(map));
  fileIndex = 0;
  for (const val of Object.values(variables as Record<string, unknown>)) {
    if (typeof File !== "undefined" && val instanceof File) {
      form.append(String(fileIndex), val);
      fileIndex += 1;
    }
  }

  const headers: Record<string, string> = {
    "Accept-Language": "tr",
    "X-Device-Id": getDeviceId(),
  };
  const token = getAccessToken();
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const res = await fetch(endpoint, { method: "POST", headers, body: form, cache: "no-store" });
  const json = (await res.json().catch(() => ({}))) as {
    data?: TResult;
    errors?: { message: string }[];
  };
  if (!res.ok || (json.errors && json.errors.length > 0)) {
    throw new ClientError(
      {
        data: json.data,
        errors: json.errors,
        status: res.status,
      } as never,
      { query, variables: opsVars },
    );
  }
  if (json.data === undefined) {
    throw new Error("işlem tamamlanamadı");
  }
  return json.data;
}

export async function restoreSession(): Promise<boolean> {
  if (getAccessToken()) {
    return true;
  }
  return tryRefresh();
}

function authHeaders(accept?: string): Record<string, string> {
  const headers: Record<string, string> = {
    "X-Device-Id": getDeviceId(),
  };
  if (accept) {
    headers.Accept = accept;
  }
  const token = getAccessToken();
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return headers;
}

/** GET /dosya/{id} — 401 olursa jeton yenileyip bir kez dener. */
export async function fetchContractFile(id: string): Promise<Blob> {
  const url = `${apiRoot()}/dosya/${encodeURIComponent(id)}`;
  let res = await fetch(url, { headers: authHeaders("application/pdf"), cache: "no-store" });
  if (res.status === 401) {
    const ok = await tryRefresh();
    if (!ok) {
      clearSession();
      redirectToLogin();
      throw new AuthExpiredError();
    }
    res = await fetch(url, { headers: authHeaders("application/pdf"), cache: "no-store" });
  }
  if (res.status === 404) {
    throw new Error("Dosya bulunamadı");
  }
  if (!res.ok) {
    throw new Error("Dosya açılamadı");
  }
  return res.blob();
}
