# Maskelenmiş metin örneği

Kaynak: yerel `LLM_DEBUG_DUMP` çıktısı (`argos-kisisel-veri.pdf`).
Kişisel veri yerine `[EPOSTA]`, `[TELEFON]`, `[TCKN]` yazılmıştır.
Ham e-posta, telefon ve TCKN bu dosyada yoktur.

```
HİZMET SÖZLEŞMESİ
Side Turizm Seyahat Acentesi ile Argos Otel arasında aşağıdaki koşullarda kontenjan sözleşmesi
akdedilmiştir.
MADDE 1 — SÖZLEŞME SÜRESİ
İşbu sözleşme 01.04.2026 tarihinde yürürlüğe girer ve 31.10.2026 tarihinde sona erer.
MADDE 4 — YETKİLİ KİŞİLER VE İLETİŞİM
Otel adına yetkili kişi: Ayşe Demir, T.C. kimlik no [TCKN]. Rezervasyon bildirimleri
[EPOSTA] adresine ve [TELEFON] numaralı telefona yapılır.
Acente adına yetkili kişi: Mehmet Yıldız. İletişim: [EPOSTA], [TELEFON].
```

Beklenen örtü sayısı bu PDF için 5'tir (2 e-posta, 2 telefon, 1 TCKN).
