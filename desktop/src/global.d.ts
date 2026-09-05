export {};

declare global {
  interface Window {
    kontrata?: {
      apiBase: string;
      getRefreshToken(): Promise<string | null>;
      setRefreshToken(token: string | null): Promise<void>;
      loadSettings?(): Promise<{
        mongoUri: string;
        llmEndpointUrl: string;
        smtpHost: string;
        smtpPort: string;
        smtpUser: string;
        smtpFrom: string;
      } | null>;
      saveSettings?(input: {
        mongoUri: string;
        llmEndpointUrl: string;
        llmToken: string;
        smtpHost: string;
        smtpPort: string;
        smtpUser: string;
        smtpPassword: string;
        smtpFrom: string;
      }): Promise<{ ok: true } | { ok: false; error: string }>;
    };
  }
}
