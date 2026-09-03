# KVKK

Model çağrılarına giden sözleşme metni `internal/mask` katmanından geçer.
Yönetici prompt'u değiştiremez bunu; e-posta, telefon (05xx cep / +90 cep veya
saha kodu; PDF satır sonuyla bölünmüş yazımlar dahil) ve 11 haneli
sayı (TCKN) her zaman `[EPOSTA]` / `[TELEFON]` / `[TCKN]` ile örtülür. Logda
yalnızca kaç desenin değiştiği yazılır, içerik yazılmaz.

## Toplanan kişisel veri

| Veri | Kaynak | Amaç |
| --- | --- | --- |
| E-posta adresi | Kayıt formu | Hesap kimliği, doğrulama ve güvenlik iletileri |
| Şifre özeti (argon2id) | Kayıt / sıfırlama | Kimlik doğrulama. Düz şifre saklanmaz |
| IP adresi | `X-Forwarded-For` veya `RemoteAddr` | Denetim izi, kaba kuvvet ve yetkisiz erişim ayrımı |
| Kullanıcı ajanı | `User-Agent` | İstemci yazılımını ayırt etmek, cihaz adı |
| Cihaz parmak izi özeti | `X-Device-Id`, `User-Agent`, `Accept-Language` | Kayıtlı cihaz; ham parmak izi saklanmaz |
| Oturum yenileme jetonu özeti | Rastgele 32 bayt | Oturum yenileme; düz jeton saklanmaz |
| MFA kodu özeti | Giriş | İki adımlı doğrulama; düz kod saklanmaz |

Pazarlama veya profilleme için kullanılmaz.

**Hukuki dayanak:** 6698 sayılı Kanun md. 5/2-c (sözleşmenin kurulması) ve
md. 5/2-f (veri sorumlusunun meşru menfaati: hesap güvenliği).

## Saklama süreleri

- Hesap ve cihaz kayıtları: hesap durduğu sürece.
- Yenileme jetonu / oturum: 7 gün (TTL).
- MFA kodu: 120 saniye (TTL).
- E-posta doğrulama kodu: 24 saat; şifre sıfırlama ve hesap silme onayı: 1 saat (TTL).
- Denetim kayıtları: 2 yıl. Süre dolunca kayıt silinir.

## Silme talebinde ne olur

`hesapSilmeIste` e-posta ile 1 saatlik onay kodu gönderir. `hesapSil` kodu
doğrulayınca aşağıdaki kayıtlar tek MongoDB işleminde (transaction) kaldırılır;
bir adım başarısız olursa hiçbiri silinmez. Bu işlem replica set gerektirir;
yerel Docker ortamı tek düğümlü `rs0` olarak çalışır.

1. Kullanıcının sözleşmeleri (`dosyaAdi` dahil; ayrı dosya deposu yok)
2. Doğrulama, sıfırlama ve silme token'ları
3. MFA kodları
4. Oturumlar
5. Cihazlar
6. Kullanıcı belgesi

Denetim kayıtları silinmez. `kullaniciId` alanı `silinmis` olur; `ipAdresi` ve
`kullaniciAjani` boşaltılır. Ardından `HESAP_SILINDI` olayı yazılır.

## Denetim kayıtları neden saklanır

KVKK md. 12 veri sorumlusuna teknik ve idari tedbirler ile işleme faaliyetlerini
kayıt altında tutma yükümlülüğü getirir. Silme talebinin kendisi de sonradan
kanıtlanabilir olmalıdır. Bu nedenle denetim belgesi durur; içindeki kişisel
veri anonimleştirilir. Ayrıntı `docs/kararlar.md` karar 8'de.

## Erişim hakkı

`verilerimiIndir` hesabın dışa aktarılabilir kopyasını JSON olarak döner.
Şifre özeti, MFA/oturum/doğrulama token'ları ve cihaz parmak izi özeti dahil
edilmez.

## Loglama

Uygulama günlüğüne (stdout) e-posta, şifre, jeton, MFA kodu, IP veya sözleşme
gövdesi yazılmaz.

## Yerel yükleme dizini

Yüklenen PDF'ler ve çıkarım dökümleri `backend/uploads/` (veya `UPLOAD_DIR`)
altında durur; dizin sürüme girmez. Bir commit'te yanlışlıkla versiyonlandı,
sonraki commit'te takip dışına alındı. Geçmişte kalan dosyalar test verisidir
(sentetik sözleşmeler); gerçek müşteri verisi değildir.
