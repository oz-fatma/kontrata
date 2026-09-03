/** Geçersiz veya kullanılmış kodda GraphQL hata vermez; false döner. */
export const verifyFailedMessage = "Doğrulama kodu geçersiz veya süresi dolmuş";

/** Yapıştırılan metinden doğrulama token'ını çıkarır (ham kod veya ?token= bağlantısı). */
export function parseVerifyInput(raw: string): string {
  const t = raw.trim();
  if (!t) {
    return "";
  }
  try {
    const u = new URL(t);
    const q = u.searchParams.get("token");
    if (q) {
      return q.trim();
    }
  } catch {
    // ham kod veya göreli sorgu
  }
  const m = t.match(/[?&]token=([^&\s#]+)/i);
  if (m) {
    try {
      return decodeURIComponent(m[1]).trim();
    } catch {
      return m[1].trim();
    }
  }
  return t;
}
