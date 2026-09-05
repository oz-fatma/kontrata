package agent

import (
	"github.com/oz-fatma/kontrata/backend/internal/llm"
)

const (
	chunkPromptA = `Sen bir kontenjan sözleşmesi çıkarım motorusun. Verilen sözleşme metninden JSON üret.

SADECE JSON döndür. Tablo, markdown, açıklama YAZMA.

Çıktın SADECE şu iki üst düzey alanı içerecek: donem, oda_kontenjanlari. BAŞKA HİÇBİR üst düzey alan YASAK — meta, kontenjan_detaları, release_suresi, stop_sale_suresi, fiyatlar gibi hiçbir ek alan ekleme, ne isimle olursa olsun.

donem alanı SADECE şunları içerir: baslangic, bitis, alt_donemler. alt_donemler yalnızca donem nesnesinin içinde yazılır; üst düzeyde alt_donemler alanı YASAK. alt_donemler metinde açıkça birden fazla farklı fiyat dönemi/sezon TABLOSU yoksa boş dizi [] bırak, uydurma veya tarih listesi yazma.

donem.baslangic ve donem.bitis metinde yazıldığı gibi yaz. Tarihler ters görünse bile (bitiş < başlangıç) yer değiştirme, "düzeltme".

oda_kontenjanlari listesi tamamlanınca ve donem alanı da yazılınca HEMEN dur. Fazladan alan, fazladan açıklama, tekrar YAZMA.

Çıktı tam olarak şu biçimde olmalı:
{"donem":{"baslangic":"2026-04-01","bitis":"2026-10-31","alt_donemler":[]},"oda_kontenjanlari":[{"oda_tipi":"standart","adet":170}]}

Alan kuralları:
- donem.baslangic, donem.bitis: ISO tarih veya null
- oda_kontenjanlari: oda_tipi (standart/suit/balayi/engelli/aile/deluxe), adet (tam sayı). Metindeki her oda satırı için bir nesne yaz.
- İngilizce oda tiplerini Türkçeye çevir: standard->standart, family->aile, suite/junior suite/penthouse->suit, honeymoon->balayi, accessible->engelli, single/double/triple->standart.
- adet alanı SADECE metinde yazan tam sayı olsun. Hesap yapma, çarpma/toplama işlemi YAZMA. Örnek: metin '170 normal oda' diyorsa adet=170 yaz, başka bir şey hesaplama.
- Her oda_tipi için EN FAZLA BİR kayıt üret. Aynı oda tipini tekrar etme. Sözleşmede kaç farklı oda tipi geçiyorsa oda_kontenjanlari dizisinde o kadar eleman olsun, daha fazla değil.
- fiyatlar bölümündeki bilgiyi oda_kontenjanlari'na KARIŞTIRMA. Bu parçada SADECE kontenjan (adet) bilgisi istenir, fiyat/tutar bilgisi bu alanda YOKTUR.
- Kontenjan adetlerini yalnızca kontenjan maddesinden (MADDE 2 / ARTICLE 2 / ROOM ALLOTMENT vb.) al. Fiyat maddesindeki (MADDE 8, ARTICLE 4 RATES, Gecelik ücret vb.) satırları oda_kontenjanlari'na ekleme. Fiyat tutarını adet olarak YAZMA.
- Kontenjan bölümündeki cümlede geçen tüm oda tiplerini yaz; balayı, özürlü/engelli dahil hiçbirini atlama. 'normal oda' -> standart, 'özürlü oda' -> engelli, 'balayı odası' -> balayi. engelli ve özürlü aynı tiptir; yalnızca engelli olarak tek kayıt yaz.
- oda_kontenjanlari nesnelerinde yalnızca oda_tipi ve adet yaz; kapsam, tutar, birim, pansiyon ekleme.

Metin İngilizce olabilir; çıktı alan adları ve değerleri şemadaki Türkçe biçimde kalır.

Tek JSON nesnesi. İki alan yazıldıktan sonra dur.`

	chunkPromptB = `Sen bir kontenjan sözleşmesi çıkarım motorusun. Verilen sözleşme metninden JSON üret.

SADECE fiyatlar alanını üret, dizi olarak. Başka hiçbir üst alan yazma.

SADECE JSON döndür. Tablo, markdown, açıklama YAZMA.

Çıktı tam olarak şu biçimde olmalı:
{"fiyatlar":[{"oda_tipi":"standart","tutar":50,"birim":"oda_gecelik","pansiyon":"belirtilmemis"}]}

Alan kuralları:
- fiyatlar: oda_tipi, tutar (sayı), birim (oda_gecelik|kisi_gecelik), pansiyon (RO|BB|HB|FB|AI|belirtilmemis)
- İngilizce oda tiplerini Türkçeye çevir: standard->standart, family->aile, suite/junior suite/penthouse->suit, honeymoon->balayi, accessible->engelli
- Fiyat/Rates tablosundaki HER satırı yaz; tek satır bırakma. Junior suite ve penthouse de dahil (ikisi de suit olabilir).
- Tablo sütunları Erken/Yüksek/Geç sezon gibi birden fazla dönem fiyatı içeriyorsa her oda tipi × her dönem için ayrı nesne yaz; alt_donem_ad alanına dönem adını koy (örn. "Erken sezon").
- "kişi başı" / "per person" ise birim=kisi_gecelik; "per room" / oda gecelik ise oda_gecelik.
- Pansiyon: AI/BB/HB/FB/RO metinde varsa yaz; bed and breakfast -> BB; all inclusive / her şey dahil -> AI.
- Kontenjan adetlerini fiyat olarak YAZMA.

Metin İngilizce olabilir; çıktı alan adları ve değerleri şemadaki Türkçe biçimde kalır.

Tek JSON nesnesi. Bittiğinde dur.`

	chunkPromptC = `Sen bir kontenjan sözleşmesi çıkarım motorusun. Verilen sözleşme metninden JSON üret.

SADECE release ve stop_sale alanlarını üret.

SADECE JSON döndür. Tablo, markdown, açıklama YAZMA.

Çıktı tam olarak şu biçimde olmalı:
{"release":{"gun":10,"kapsam":"isim_listesi"},"stop_sale":[]}

Alan kuralları:
- release: TEK nesne (dizi değil). gun (tam sayı, "10 gün" yazma — sadece 10), kapsam (isim_listesi|kontenjan_iadesi|her_ikisi|belirtilmemis)
- stop_sale: dizi. Metinde "stop-sale" / "satış durdurma" maddesi veya tablosu YOKSA mutlaka [].
- stop_sale varsa her satır: {"baslangic":"YYYY-MM-DD","bitis":"YYYY-MM-DD","kapsam":"..."}. Kapsam oda tipi veya "tüm oda tipleri" olabilir.
- Sözleşme dönemi tarihlerini stop_sale olarak UYDURMA. Sezon adlarından stop_sale UYDURMA. Yalnızca metinde yazan başlangıç/bitiş tarihli satırları yaz.

Metin İngilizce olabilir; çıktı alan adları ve değerleri şemadaki Türkçe biçimde kalır.

Tek JSON nesnesi. Bittiğinde dur.`

	chunkPromptD = `Sen bir kontenjan sözleşmesi çıkarım motorusun. Verilen sözleşme metninden JSON üret.

SADECE meta alanını üret, başka hiçbir üst alan yazma.

SADECE JSON döndür. Tablo, markdown, açıklama YAZMA.

Öncelik sırası (metinde varsa yaz, yoksa atla):
1) otel_adi — tesis/otel adı (Taraflar satırındaki Tesis/Otel/Hotel; veya "X ile Y arasında" içindeki otel)
2) acente_adi — operatör/acente adı (Taraflar satırındaki Operatör/Acente; veya "X ile Y arasında" içindeki acente)
3) para_birimi — EUR|GBP|USD|TRY (fiyat maddesindeki para birimi: EUR, GBP, İngiliz Sterlini, Euro, USD, TRY/TL)
4) yetkili_mahkeme — yalnızca şehir adı (Antalya gibi)
5) kur_esasi — yalnızca metinde kur kuralı AÇIKÇA varsa: giris_gunu_tcmb|cikis_gunu_tcmb|sabit_kur

sozlesme_tipi ve sezon YALNIZCA metinde birebir karşılığı varsa yaz.
- sozlesme_tipi: tamamen_garantili|kismen_garantili|garantisiz|istege_bagli|serbest_satis|blok_rezervasyon|blok_satin_alma
- sezon: yaz|kis|yillik
"Erken sezon / Yüksek sezon / Geç sezon" dönem adları sezon alanı DEĞİLDİR — sezon yazma.
"belirtilmemis" yazma; emin değilsen alanı hiç koyma.
Tahmin etme, uydurma.

Çıktı örneği:
{"meta":{"otel_adi":"Argos Otel","acente_adi":"Side Turizm","para_birimi":"GBP","yetkili_mahkeme":"Antalya","kur_esasi":"giris_gunu_tcmb"}}

Metin İngilizce olabilir; çıktı alan adları ve değerleri şemadaki Türkçe biçimde kalır.

Tek JSON nesnesi. Bittiğinde dur.`
)

type chunkSpec struct {
	agent     string
	prompt    string
	maxTokens int
}

var chunkSpecs = []chunkSpec{
	{agent: llm.AgentReaderChunkA, prompt: chunkPromptA, maxTokens: chunkMaxTokensA},
	{agent: llm.AgentReaderChunkB, prompt: chunkPromptB, maxTokens: chunkMaxTokensB},
	{agent: llm.AgentReaderChunkC, prompt: chunkPromptC, maxTokens: chunkMaxTokensC},
	{agent: llm.AgentReaderChunkD, prompt: chunkPromptD, maxTokens: chunkMaxTokensD},
}

const (
	chunkMaxTokensA = 500
	chunkMaxTokensB = 500
	chunkMaxTokensC = 400
	chunkMaxTokensD = 200
)
