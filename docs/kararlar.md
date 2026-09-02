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

## 8. Hesap silmede denetim kaydı
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: KVKK md. 7 silme hakkını tanır; md. 12 veri sorumlusuna işleme faaliyetlerini kayıt altında tutma yükümlülüğü getirir. Silme talebinin kendisi de kanıtlanabilir olmalıdır.
Karar: Hesap silinince sözleşmeler, token'lar, MFA kodları, oturumlar, cihazlar ve kullanıcı belgesi tek MongoDB işleminde (transaction) kaldırılır; bu yüzden sunucu replica set (veya mongos) olmalıdır. Yerel Docker ve CI bu yüzden tek düğümlü replica set (`rs0`) olarak çalışır. `denetim_kayitlari` silinmez; `kullaniciId` değeri `silinmis` olur, `ipAdresi` ve `kullaniciAjani` boşaltılır. İşlem `HESAP_SILINDI` olayıyla kapanır.
Sonuç: Kısmi silme olmaz. Denetim izi kişisel veriden arındırılmış biçimde kalır. Replica setsiz MongoDB'de silme işlemi reddedilir.

## 9. CI MongoDB replica set
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Hesap silme transaction replica set ister. GitHub Actions `services` bloğu konteynere `mongod --replSet` argümanı geçirmez; `mongo:8` servisi tek düğüm kalır. `mongodb/mongodb-atlas-local` replica set hazır gelir ve servis olarak kullanılabilir, ancak Atlas Search süreci taşır ve yerel `docker-compose` imajından (`mongo:8`) sapar.
Karar: CI, yerel ortamla aynı `mongo:8 --replSet rs0` sürecini iş adımında `docker run` ile başlatır. Testler `rs.status()` ve birincil olana kadar beklemeden başlamaz.
Sonuç: Yerel ve CI aynı motoru ve replica set adını (`rs0`) kullanır. GHA `services` bloğunda Mongo yoktur.

## 10. Organizasyon kapsamı ve roller
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Bireysel hesap kendi sözleşmesinin sahibidir; kurumsal hesapta sözleşme kuruma aittir. Çok kiracılı sistemlerde en sık hata başka kiracının belgesini kimlik ile okumaktır.
Karar: `sozlesmeler` koleksiyonuna `organizasyonId` yazılır. Kurumsal listede süzgeç yalnızca bu alandır; bireyselde `kullaniciId` ve boş `organizasyonId`. Resolver, kaydı getirmeden önce kullanıcının organizasyonuyla karşılaştırır; eşleşmezse `null` döner (yokmuş gibi). Rol yetkisi (`SAHIP` / `YONETICI` / `GORUNTULEYICI`) arayüz gizlemesine bırakılmaz, resolver'da kesilir. Mevcut belgeler açılışta kurumsal kullanıcının `organizasyonId` değeriyle doldurulur.
Sonuç: Üye daveti 7 günlük TTL kodla gelir; sahip başka üye varken hesabını silemez, önce devir veya `organizasyonSil` gerekir.

## 11. Sentetik sözleşme verisi şablonla üretilir
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Okuyucu agent'ı eğitmek için yüzlerce kontenjan sözleşmesi gerekir; gerçek otel kontratları tesiste kalır ve eğitim kümesine alınamaz. Tek kamuya açık örnek MEGEP Argos sözleşmesidir.
Karar: `ml/generate.py` şemadan rastgele geçerli nesne üretir, metni LLM'siz şablonla yazar. MEGEP örneği (`ornek-argos.json`) eğitim/doğrulama jsonl dosyalarına karışmaz; tutulur ki model tek gerçek örneği ezberlemesin. Kasıtlı gürültü (eksik alan, tarih çelişkisi, fiyat–kontenjan uyuşmazlığı) JSON Schema'yı bozmaz.
Sonuç: Üretim deterministiktir (`--seed`). Üretilen `ml/data/*.jsonl` sürüme girmez.

## 12. Go tanımlayıcıları İngilizce
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Go iç kodu ile GraphQL/veritabanı katmanı farklı diller kullanıyordu, tutarsızdı.
Karar: Go tanımlayıcıları İngilizce; GraphQL şeması, MongoDB alanları, yorumlar ve kullanıcıya dönen metinler Türkçe.
Sonuç: Kod uluslararası okunabilir, domain dili sektör terimleriyle örtüşmeye devam ediyor.


