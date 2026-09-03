import type { SozlesmeDurumu as ContractStatus } from "@/generated/graphql";
import { SozlesmeDurumu } from "@/lib/enums";

export function formatUserAgent(ua: string | null | undefined): string {
  if (!ua?.trim()) {
    return "Bilinmeyen cihaz";
  }
  let browser = "Tarayıcı";
  if (/Electron\//i.test(ua)) {
    browser = "Electron";
  } else if (/Edg\//i.test(ua)) {
    browser = "Edge";
  } else if (/Chrome\//i.test(ua)) {
    browser = "Chrome";
  } else if (/Firefox\//i.test(ua)) {
    browser = "Firefox";
  } else if (/Safari\//i.test(ua)) {
    browser = "Safari";
  }

  let os = "bilinmeyen sistem";
  if (/iPhone|iPad|iOS/i.test(ua)) {
    os = "iOS";
  } else if (/Android/i.test(ua)) {
    os = "Android";
  } else if (/Mac OS X|Macintosh/i.test(ua)) {
    os = "macOS";
  } else if (/Windows/i.test(ua)) {
    os = "Windows";
  } else if (/Linux/i.test(ua)) {
    os = "Linux";
  }

  return `${browser} / ${os}`;
}

export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) {
    return "—";
  }
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return "—";
  }
  return new Intl.DateTimeFormat("tr-TR", {
    dateStyle: "short",
    timeStyle: "short",
  }).format(d);
}

export function formatPeriod(
  start: string | null | undefined,
  end: string | null | undefined,
): string {
  if (!start && !end) {
    return "—";
  }
  return `${start ?? "—"} – ${end ?? "—"}`;
}

export function statusLabel(durum: ContractStatus): string {
  switch (durum) {
    case SozlesmeDurumu.Yuklendi:
      return "Yüklendi";
    case SozlesmeDurumu.Isleniyor:
      return "İşleniyor";
    case SozlesmeDurumu.IncelenmeyiBekliyor:
      return "İncelenmeyi bekliyor";
    case SozlesmeDurumu.Onaylandi:
      return "Onaylandı";
    case SozlesmeDurumu.Hata:
      return "Hata";
    default:
      return durum;
  }
}

export type StatusTone = "muted" | "blue" | "yellow" | "green" | "red";

export function statusTone(durum: ContractStatus): StatusTone {
  switch (durum) {
    case SozlesmeDurumu.Isleniyor:
      return "blue";
    case SozlesmeDurumu.IncelenmeyiBekliyor:
      return "yellow";
    case SozlesmeDurumu.Onaylandi:
      return "green";
    case SozlesmeDurumu.Hata:
      return "red";
    default:
      return "muted";
  }
}

export function confidenceLabel(
  page: number | null | undefined,
  score: number | null | undefined,
): string | null {
  const parts: string[] = [];
  if (typeof page === "number") {
    parts.push(`s.${page}`);
  }
  if (typeof score === "number") {
    parts.push(`%${Math.round(score * 100)}`);
  }
  return parts.length ? parts.join(" · ") : null;
}

export function missingField(): string {
  return "Sözleşmede madde yok";
}

export function roleLabel(rol: string): string {
  switch (rol) {
    case "SAHIP":
      return "Sahip";
    case "YONETICI":
      return "Yönetici";
    case "GORUNTULEYICI":
      return "Görüntüleyici";
    default:
      return rol;
  }
}

export function enumLabel(value: string | null | undefined): string {
  if (!value) {
    return missingField();
  }
  const map: Record<string, string> = {
    TAMAMEN_GARANTILI: "Tamamen garantili",
    KISMEN_GARANTILI: "Kısmen garantili",
    GARANTISIZ: "Garantisiz",
    ISTEGE_BAGLI: "İsteğe bağlı",
    SERBEST_SATIS: "Serbest satış",
    BLOK_REZERVASYON: "Blok rezervasyon",
    BLOK_SATIN_ALMA: "Blok satın alma",
    BELIRTILMEMIS: "Belirtilmemiş",
    YAZ: "Yaz",
    KIS: "Kış",
    YILLIK: "Yıllık",
    GIRIS_GUNU_TCMB: "Giriş günü TCMB",
    CIKIS_GUNU_TCMB: "Çıkış günü TCMB",
    SABIT_KUR: "Sabit kur",
    ODA_GECELIK: "Oda gecelik",
    KISI_GECELIK: "Kişi gecelik",
    ISIM_LISTESI: "İsim listesi",
    KONTENJAN_IADESI: "Kontenjan iadesi",
    HER_IKISI: "Her ikisi",
    YAZILI: "Yazılı",
    FAKS: "Faks",
    EPOSTA: "E-posta",
    SISTEM: "Sistem",
  };
  return map[value] ?? value;
}
