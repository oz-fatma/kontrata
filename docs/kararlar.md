# Kararlar

## 2026-09-01 — Veritabanı: MongoDB

Sözleşme yapısı operatörden operatöre değişken olduğu için doküman modeli seçildi.

## 2026-09-02 — Go hedef sürümü 1.25

Yerelde Go 1.26 kurulu, ancak golangci-lint henüz 1.26 ile derlenmiş sürüm yayınlamadı; CI lint işi bu yüzden başarısız oluyor.

Karar: `go.mod` hedefi 1.25'e sabitlendi.

Sonuç: Go 1.26 özellikleri kullanılamaz; araç zinciri yakaladığında yükseltilecek.
