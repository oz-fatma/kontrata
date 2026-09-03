# KVKK

Bu belge Kontrata'nın kişisel veri envanterini, saklama sürelerini, maskeleme
katmanını ve silme/anonimleştirme davranışını açıklar. Hukuki dayanak:
6698 sayılı Kanun md. 5/2-c (sözleşmenin kurulması) ve md. 5/2-f (hesap
güvenliği için meşru menfaat).

Sözleşme verisi ve kişisel veri hiçbir zaman uygulama günlüğüne yazılmaz.
Model çağrılarına giden metin önce `internal/mask` katmanından geçer.
Pazarlama veya profilleme yapılmaz.

## Toplanan kişisel veri

| Veri | Amaç | Süre | Nerede |
| --- | --- | --- | --- |
| E-posta adresi | Hesap kimliği; doğrulama, MFA, silme onayı ve davet iletileri | Hesap durduğu sürece | MongoDB `kullanicilar.eposta` |
| Şifre özeti (argon2id PHC) | Kimlik doğrulama. Düz şifre saklanmaz | Hesap durduğu sürece | `kullanicilar.sifreHash` |
| IP adresi (`X-Forwarded-For` / `RemoteAddr`) | Denetim izi, kaba kuvvet ve yetkisiz erişim ayrımı | Hesap durduğu sürece cihaz/oturumda; denetimde 2 yıl (anonimleşir) | `oturumlar`, `cihazlar`, `denetim_kayitlari` |
| Kullanıcı ajanı | İstemci yazılımını ayırt etmek, cihaz adı | Cihaz kaydı durduğu sürece; denetimde 2 yıl (anonimleşir) | `cihazlar`, `oturumlar`, `denetim_kayitlari` |
| Cihaz parmak izi özeti (`X-Device-Id` + UA + dil) | Kayıtlı cihaz. Ham parmak izi saklanmaz | Hesap durduğu sürece | `cihazlar` |
| Oturum yenileme jetonu özeti | Oturum yenileme. Düz jeton saklanmaz | 7 gün (TTL) | `oturumlar` |
| MFA kodu özeti | İki adımlı giriş. Düz kod saklanmaz | 120 saniye (TTL) | `mfa_kodlari` |
| Doğrulama / sıfırlama / silme kodu özeti | E-posta doğrulama, şifre sıfırlama, hesap silme onayı | 24 saat / 1 saat (TTL) | `dogrulama_tokenlari` |
| Sözleşmelerdeki üçüncü kişi verileri (yetkili adı, e-posta, telefon, TCKN, otel/acente unvanı) | Kontenjan sözleşmesini yapılandırmak; tesiste kalır | Hesap/organizasyon durduğu sürece | `sozlesmeler` ve `UPLOAD_DIR` PDF'leri. LLM'e yalnızca maskelenmiş metin gider |

E-posta doğrulama kodu 24 saat, şifre sıfırlama ve hesap silme onayı 1 saat geçerlidir.

## Maskeleme katmanı

Model çağrısından önce `internal/mask` çalışır. Yönetici prompt'u veya ayar
paneli bunu kapatamaz; her Okuyucu ve Denetçi isteğinden önce zorunludur.

| Desen | Örtü | Neden kapatılamaz |
| --- | --- | --- |
| E-posta (`ad@alan.tld`) | `[EPOSTA]` | Operatör sözleşmesinde yetkili iletişim bilgisi sık geçer; modele gitmesi KVKK md. 12 tedbirini zayıflatır |
| Ulusal cep (`05xx`, ayırıcı ve PDF satır sonu dahil) | `[TELEFON]` | Rezervasyon bildirim numarası sözleşmenin gövdesindedir |
| `+90` + 10 hane (cep veya saha kodu; satır sonu ayırıcı sayılır) | `[TELEFON]` | PDF metin çıkarımı numarayı satır ortasında böler; dar desen kaçırır |
| 11 haneli sayı | `[TCKN]` | Yetkili kişi kimlik no nadiren yazılır; yanlış pozitif (fiyat) göze alınır |

Logda yalnızca kaç desenin değiştiği yazılır, örtülen içerik yazılmaz.
Örnek (dump'tan alınmış, kişisel veri yok): [`ornek-maskelenmis-metin.md`](ornek-maskelenmis-metin.md).

## Silme talebinde ne olur

`hesapSilmeIste` e-posta ile 1 saatlik onay kodu gönderir. `hesapSil` kodu
doğrulayınca aşağıdaki kayıtlar tek MongoDB işleminde (transaction) kaldırılır;
bir adım başarısız olursa hiçbiri silinmez. Replica set gerekir.

1. Organizasyona ait sözleşmeler (`sozlesmeler`)
2. Prompt sürümleri (`prompt_surumleri`) ve çalışma ayarları (`ayarlar`)
3. LLM izleme kayıtları (`llm_cagrilari`) — zaten metin içermez
4. Davetler
5. Doğrulama, sıfırlama ve silme token'ları
6. MFA kodları
7. Oturumlar
8. Cihazlar
9. Kullanıcı belgesi (`kullanicilar`)
10. Organizasyon belgesi
11. Transaction commit olduktan sonra `UPLOAD_DIR` altındaki PDF'ler (UUID ad)

Sahip, başka üye varken hesabını silemez; önce devir veya organizasyon silme gerekir.
Uçtan uca doğrulama: `TestKVKKSilmeZinciri`.

### Neyin anonimleştirildiği ve neden

`denetim_kayitlari` **silinmez**. `kullaniciId` değeri `silinmis` olur;
`ipAdresi` ve `kullaniciAjani` boşaltılır. Ardından `HESAP_SILINDI` olayı yazılır.

Silme talebinin kendisi sonradan kanıtlanabilir olmalıdır. Kayıt durur; içindeki
kişisel veri kalkar.

## Denetim kayıtlarının saklanma gerekçesi

KVKK md. 12 veri sorumlusuna teknik ve idari tedbirler ile işleme faaliyetlerini
kayıt altında tutma yükümlülüğü getirir. Kimlik doğrulama, yetkilendirme ve
silme işlemleri bu koleksiyona yazılır. Süre 2 yıl; TTL dolunca kayıt silinir.
Ayrıntı `kararlar.md` karar 8.

## Erişim hakkı

`verilerimiIndir` hesabın dışa aktarılabilir kopyasını JSON olarak döner.
Şifre özeti, MFA/oturum/doğrulama token'ları, cihaz parmak izi özeti ve MFA
kodu dahil edilmez. Doğrulama: `TestKVKKVeriIhraci`.

## Veri nereye gider

Veri hiçbir bulut depolama, analitik veya LLMOps hizmetine gitmez. MongoDB
tesiste (veya geliştirmede yerel Docker) çalışır. PDF'ler `UPLOAD_DIR` içindedir.

**Tek istisna:** HuggingFace Inference Endpoint üzerindeki çıkarım. Modele giden
metin maskelenmiştir; `llm_cagrilari` süre, uç adı, karakter sayısı ve hata tipi
tutar, gövde tutmaz. Langfuse kullanılmaz (`kararlar.md` karar 28).

## Loglama

Stdout'a e-posta, şifre, jeton, MFA kodu, IP veya sözleşme gövdesi yazılmaz.
`LLM_DEBUG_DUMP` yalnızca yerel hata ayıklamada maskelenmiş metin ve ham model
çıktısını diske yazar; üretimde kapalıdır.
