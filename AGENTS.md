# Kontrata

Otellerin tur operatörlerinden aldığı kontenjan sözleşmelerini okuyup
yapılandırılmış veriye çeviren masaüstü uygulaması. İki agent çalışır:
Okuyucu sözleşme PDF'ini şemaya çevirir, Denetçi çıkan veride çelişki,
eksik madde ve riskli şart arar. Sözleşme verisi tesisten çıkmaz —
masaüstü tercihinin sebebi budur.

## Teknoloji (değiştirilemez)
- Backend: Go 1.25, Chi router, gqlgen (GraphQL)
- Veritabanı: MongoDB
- Arayüz: Next.js + TypeScript (static export)
- Masaüstü: Electron (Windows + macOS)
- Model: HuggingFace Inference Endpoint üzerinde fine-tune edilmiş model
- İzleme: Langfuse

## Go modül yolu
github.com/oz-fatma/kontrata/backend

## Dizin yapısı
backend/   Go servisi
web/       Next.js arayüz
desktop/   Electron kabuk
ml/        sentetik veri üretimi, fine-tune, değerlendirme
docs/      mimari kararlar ve KVKK notları

## Kod kuralları
- Go: standart proje düzeni, iç paketler internal/ altında
- GraphQL şeması elle yazılmaz, gqlgen ile üretilir
- Hata mesajlarında kullanıcıya dönen metin ile loga yazılan ayrıntı ayrılır
- Dosya ve dizin adları ASCII: Türkçe karakter kullanma

## Dil kuralları

Go kodu:
- Fonksiyon, metot, tip, alan, değişken, sabit adları: İngilizce
- Dosya adları: İngilizce (contract.go, session.go, device.go)
- Test fonksiyonları ve dosyaları: İngilizce (TestAuthFlow,
  auth_flow_test.go)
- Hata değişkenleri: İngilizce (ErrInvalidRefreshToken)

Python kodu (ml/):
- Fonksiyon, sınıf, değişken adları: İngilizce
- Dosya adları: İngilizce

TypeScript kodu (web/, desktop/):
- Bileşen, fonksiyon, tip, değişken adları: İngilizce
- Dosya adları: İngilizce

Türkçe kalanlar:
- GraphQL şeması ve alan adları (domain dili)
- MongoDB koleksiyon adları ve bson etiketleri
- Kod içi yorumlar
- Log mesajları
- Kullanıcıya dönen hata metinleri ve arayüz metinleri
- docs/ altındaki tüm belgeler

Kural: makine tarafı İngilizce, insan tarafı Türkçe.

## Güvenlik ve uyum kuralları
- Sözleşme verisi ve kişisel veri hiçbir zaman loglara yazılmaz
- Model çağrılarına giden metin önce maskeleme katmanından geçer
- Kimlik doğrulama, yetkilendirme ve silme işlemleri denetim kaydına yazılır
- Yetki kontrolü resolver seviyesinde yapılır, arayüzde gizlemek yeterli değil

## Çalışma biçimi
- Her mimari karar docs/kararlar.md dosyasına tarihiyle yazılır
- Bir aşama bitmeden sonrakine geçilmez
- İstenmeyen dosya, bağımlılık veya soyutlama ekleme
