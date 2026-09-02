# KVKK

Kontrata sözleşme verisini tesiste tutar; kişisel veri uygulama günlüklerine yazılmaz.
Aşağıdaki kayıtlar kimlik işlemlerinin güvenliği için tutulur.

## Denetim kaydındaki IP adresi ve kullanıcı ajanı

`denetim_kayitlari` belgelerinde `ipAdresi` ve `kullaniciAjani` saklanır.

**Neden toplanır:** Kayıt, e-posta doğrulama ve şifre sıfırlama gibi kimlik işlemlerinde
yetkisiz erişim, hesap ele geçirme ve kaba kuvvet denemelerini ayırt etmek için.
IP, isteği gönderen ağı; kullanıcı ajanı istemci yazılımını gösterir. Pazarlama
veya profilleme için kullanılmaz. Uygulama loguna (stdout) yazılmaz.

**Hukuki dayanak:** 6698 sayılı Kanun md. 5/2-f (veri sorumlusunun meşru menfaati:
hesap güvenliği ve denetim izi).

**Saklama süresi:** 2 yıl. Süre dolunca kayıt silinir. IP kişisel veridir; süre
dolmadan silme talebi, güvenlik izinin bütünlüğünü zedelemediği ölçüde değerlendirilir.

**Kaynak:** HTTP `RemoteAddr`; istekte `X-Forwarded-For` varsa listedeki ilk adres
kullanılır. Kullanıcı ajanı `User-Agent` başlığından alınır.
