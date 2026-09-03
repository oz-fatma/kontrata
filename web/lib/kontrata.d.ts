export {};

declare global {
  interface Window {
    kontrata?: {
      apiBase: string;
      getRefreshToken(): Promise<string | null>;
      setRefreshToken(token: string | null): Promise<void>;
    };
  }
}
