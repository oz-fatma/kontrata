# Kararlar

## 1. Masaüstü uygulaması tercihi
Tarih: 2026-09-01
Durum: kabul edildi
Bağlam: Otel-tur operatörü sözleşmelerinde fiyat ve kontenjan bilgisi ticari sırdır; oteller bu veriyi bulut hizmetine yüklemek istemez. Rakip ürünler bulut SaaS olduğu için bu segmente giremiyor.
Karar: Electron ile masaüstü uygulaması, veri tesiste kalır.
Sonuç: Dağıtım ve güncelleme daha zor, kod imzalama gerekir; buna karşılık gizlilik doğrudan satış argümanı olur.

## 2. MongoDB tercihi
Tarih: 2026-09-01
Durum: kabul edildi
Bağlam: Her tur operatörünün sözleşme formatı farklı, çıkarılan alan kümesi sabit değil.
Karar: Doküman veritabanı.
Sonuç: Şema esnekliği kazanıldı, ilişkisel bütünlük garantileri uygulama katmanına taşındı.

## 3. GraphQL tercihi
Tarih: 2026-09-01
Durum: kabul edildi
Bağlam: Sözleşme detay ekranı değişken alan kümeleri çekiyor, liste ekranı ise az alan.
Karar: gqlgen ile şemadan kod üretimi.
Sonuç: Aşırı veri çekimi önlendi; buna karşılık REST'e göre önbellekleme ve hata yönetimi daha karmaşık.

## 4. Sağlık kontrolünde dayanıklılık
Tarih: 2026-09-01
Durum: kabul edildi
Bağlam: MongoDB erişilemez olduğunda servisin davranışı tanımlanmalıydı.
Karar: Açılışta çökme yok; `/healthz` degraded durumu ve HTTP 503 döner, uygulama ayakta kalır.
Sonuç: Kısmi hizmet mümkün, izleme sistemleri durumu ayırt edebilir.

## 5. Go hedef sürümü 1.25
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Yerelde Go 1.26 kurulu, ancak golangci-lint henüz 1.26 ile derlenmiş sürüm yayınlamadı; CI lint işi bu yüzden başarısız oluyor.
Karar: `go.mod` hedefi 1.25'e sabitlendi.
Sonuç: Go 1.26 özellikleri kullanılamaz; araç zinciri yakaladığında yükseltilecek.

## 6. Kayıt ve e-posta doğrulama
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Kullanıcı hesabı e-posta ile doğrulanacak; hesap varlığı yanıt veya günlüklerden sızmamalı, şifreler geri döndürülemez saklanmalı.
Karar: Şifre argon2id (OWASP varsayılan maliyet) ile PHC biçiminde tutulur. Doğrulama kodu 32 bayt rastgele değer, veritabanında SHA-256. `kayitOl` ve `dogrulamaTekrarGonder` e-posta kayıtlı olsa da aynı yanıtı verir. Denetim kaydına kimlik işlemi yazılır; şifre, token ve e-posta loglanmaz.
Sonuç: GraphQL yüzeyi kullanıcı veya oturum token'ı dönmez; geliştirmede ConsoleMailer alıcıyı maskeleyerek iletiyi basar.

## 7. MFA ve oturum jetonları
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Masaüstü istemcinin API'ye kimlik doğrulaması gerekir; şifre tek başına oturum açmamalı.
Karar: Girişte 6 haneli MFA (120 sn, 5 deneme). Erişim jetonu JWT HS256 15 dk; yenileme jetonu 32 bayt rastgele, 7 gün, rotasyonlu. JWT'de e-posta yok. Sözleşme alanları, `cikisYap` ve `oturumlarim` `@auth` ister. `jetonYenile` erişim jetonu istemez; kimlik yenileme jetonunun kendisiyle doğrulanır. `girisYap`, `mfaDogrula`, `kayitOl` ve şifre sıfırlama alanları da `@auth` dışındadır.
Sonuç: `JWT_SECRET` zorunlu; şifre sıfırlama tüm oturumları iptal eder. Süresi dolmuş erişim jetonu yenilemeyi engellemez.


