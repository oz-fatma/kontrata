import { contextBridge, ipcRenderer } from "electron";

const apiArg = process.argv.find((a) => a.startsWith("--kontrata-api="));
const apiBase = apiArg ? apiArg.slice("--kontrata-api=".length) : "http://127.0.0.1:17890";

export type SetupFormInput = {
  mongoUri: string;
  llmEndpointUrl: string;
  llmToken: string;
  smtpHost: string;
  smtpPort: string;
  smtpUser: string;
  smtpPassword: string;
  smtpFrom: string;
};

export type SetupFormSnapshot = Omit<SetupFormInput, "llmToken" | "smtpPassword">;

contextBridge.exposeInMainWorld("kontrata", {
  apiBase,
  getRefreshToken: (): Promise<string | null> => ipcRenderer.invoke("token:get"),
  setRefreshToken: (token: string | null): Promise<void> => ipcRenderer.invoke("token:set", token),
  loadSettings: (): Promise<SetupFormSnapshot | null> => ipcRenderer.invoke("setup:load"),
  saveSettings: (
    input: SetupFormInput,
  ): Promise<{ ok: true } | { ok: false; error: string }> => ipcRenderer.invoke("setup:save", input),
});
