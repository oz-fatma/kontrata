export const HesapTipi = {
  Bireysel: "BIREYSEL",
  Kurumsal: "KURUMSAL",
} as const;

export const Rol = {
  Goruntuleyici: "GORUNTULEYICI",
  Sahip: "SAHIP",
  Yonetici: "YONETICI",
} as const;

export const SozlesmeDurumu = {
  Yuklendi: "YUKLENDI",
  Isleniyor: "ISLENIYOR",
  IncelenmeyiBekliyor: "INCELENMEYI_BEKLIYOR",
  Onaylandi: "ONAYLANDI",
  Hata: "HATA",
} as const;

export const BulguOnemi = {
  Kritik: "KRITIK",
  Uyari: "UYARI",
  Bilgi: "BILGI",
} as const;

export const BulguKaynagi = {
  Kural: "KURAL",
  Model: "MODEL",
} as const;

export const PromptTipi = {
  Okuyucu: "OKUYUCU",
  Denetci: "DENETCI",
} as const;
