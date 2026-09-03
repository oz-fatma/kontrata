# Test sözleşmeleri

Okuyucu ve Denetçi agent'ları denemek için üç PDF. Metin katmanı var,
OCR gerektirmez. `make_pdfs.py` ile yeniden üretilebilir.

## argos-megep.pdf — bilinen doğru çıktı

MEB MEGEP "Paket Tur Üretimi" modülündeki örnek kontenjan sözleşmesi.
Beklenen çıktı `ml/schema/ornek-argos.json` dosyasında hazır; Okuyucu
agent'ın çıktısı bununla birebir karşılaştırılabilir.

Beklenenler:
- Dönem: 2026-04-01 – 2026-10-31
- Kontenjan: 170 standart, 20 suit, 5 balayi, 1 engelli
- Release: 10 gün, isim listesi
- Para birimi GBP, kur esası giriş günü TCMB, yetkili mahkeme Antalya
- Stop-sale: yok (2011 tarihli sözleşme, o kavram geçmiyor)

Denetçi'nin bulması gerekenler:
- Fiyat tablosundaki oda tipleri (tek/iki/üç kişilik) kontenjan
  tablosundakilerle (normal/suit/balayı/özürlü) örtüşmüyor
- Stop-sale maddesi yok

## tui-2026-yaz.pdf — modern operatör kontratı, temiz

İki sayfa. Güncel operatör terminolojisi: release süresi, stop-sale
aralıkları, alt fiyat dönemleri, çocuk politikası, ödeme koşulları.

Beklenenler:
- Dönem: 2026-04-15 – 2026-10-25
- Alt dönemler: Erken sezon, Yüksek sezon, Geç sezon
- Kontenjan: 240 standart, 45 aile, 30 deluxe, 12 suit
- Fiyatlar: 12 satır (4 oda tipi x 3 dönem), kişi başı gecelik, AI, EUR
- Release: 14 gün, kontenjan iadesi
- Stop-sale: 2 aralık (10-18 Temmuz tüm tipler, 20-24 Ağustos suit)

Denetçi'nin bulguları az olmalı; bu sözleşme tutarlı.

Zorluk noktaları: virgüllü ondalık (48,00), Türkçe ay adı
("15 Nisan 2026"), çok sütunlu fiyat tablosu, iki sayfaya yayılma.

## coral-bozuk.pdf — kasıtlı hatalı

İngilizce, tek sayfa. Denetçi agent'ı sınamak için üç hata gömülü:

1. **Çelişkili tarih** — sözleşme 01.05.2026'da başlayıp 20.04.2026'da
   bitiyor; bitiş başlangıçtan önce.
2. **Fiyat–kontenjan uyuşmazlığı** — fiyat tablosunda "Junior suite" ve
   "Penthouse" var, kontenjan tablosunda yok.
3. **Stop-sale maddesi yok** — satış durdurma koşulu tanımlanmamış.

Ayrıca release ifadesi belirsiz: "approximately 10 days" — kesin değer
yok, düşük güven skoru beklenir.

Okuyucu'nun İngilizce oda tiplerini (standard, family) şemadaki Türkçe
değerlere (standart, aile) eşlemesi de burada test edilir.
