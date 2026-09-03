export {};

declare global {
  interface Window {
    kontrata?: {
      apiBase: string;
      getRefreshToken(): Promise<string | null>;
      setRefreshToken(token: string | null): Promise<void>;
      saveSettings?(input: {
        mongoUri: string;
        llmEndpointUrl: string;
        llmToken: string;
      }): Promise<{ ok: true } | { ok: false; error: string }>;
    };
  }
}
