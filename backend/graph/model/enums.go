package model

// GraphQL enum'larının depolama değerleri kontrat.json ile aynıdır.
// GraphQL yüzeyi SCREAMING_SNAKE_CASE kullanır; eşleme gqlgen.yml'dedir.

type SozlesmeTipi string

const (
	SozlesmeTipiTamamenGarantili SozlesmeTipi = "tamamen_garantili"
	SozlesmeTipiKismenGarantili  SozlesmeTipi = "kismen_garantili"
	SozlesmeTipiGarantisiz       SozlesmeTipi = "garantisiz"
	SozlesmeTipiIstegeBagli      SozlesmeTipi = "istege_bagli"
	SozlesmeTipiSerbestSatis     SozlesmeTipi = "serbest_satis"
	SozlesmeTipiBlokRezervasyon  SozlesmeTipi = "blok_rezervasyon"
	SozlesmeTipiBlokSatinAlma    SozlesmeTipi = "blok_satin_alma"
	SozlesmeTipiBelirtilmemis    SozlesmeTipi = "belirtilmemis"
)

type Sezon string

const (
	SezonYaz           Sezon = "yaz"
	SezonKis           Sezon = "kis"
	SezonYillik        Sezon = "yillik"
	SezonBelirtilmemis Sezon = "belirtilmemis"
)

type KurEsasi string

const (
	KurEsasiGirisGunuTcmb KurEsasi = "giris_gunu_tcmb"
	KurEsasiCikisGunuTcmb KurEsasi = "cikis_gunu_tcmb"
	KurEsasiSabitKur      KurEsasi = "sabit_kur"
	KurEsasiBelirtilmemis KurEsasi = "belirtilmemis"
)

type Pansiyon string

const (
	PansiyonRo            Pansiyon = "RO"
	PansiyonBb            Pansiyon = "BB"
	PansiyonHb            Pansiyon = "HB"
	PansiyonFb            Pansiyon = "FB"
	PansiyonAi            Pansiyon = "AI"
	PansiyonBelirtilmemis Pansiyon = "belirtilmemis"
)

type FiyatBirimi string

const (
	FiyatBirimiOdaGecelik  FiyatBirimi = "oda_gecelik"
	FiyatBirimiKisiGecelik FiyatBirimi = "kisi_gecelik"
)

type ReleaseKapsami string

const (
	ReleaseKapsamiIsimListesi     ReleaseKapsami = "isim_listesi"
	ReleaseKapsamiKontenjanIadesi ReleaseKapsami = "kontenjan_iadesi"
	ReleaseKapsamiHerIkisi        ReleaseKapsami = "her_ikisi"
	ReleaseKapsamiBelirtilmemis   ReleaseKapsami = "belirtilmemis"
)

type BildirimYontemi string

const (
	BildirimYontemiYazili        BildirimYontemi = "yazili"
	BildirimYontemiFaks          BildirimYontemi = "faks"
	BildirimYontemiEposta        BildirimYontemi = "eposta"
	BildirimYontemiSistem        BildirimYontemi = "sistem"
	BildirimYontemiBelirtilmemis BildirimYontemi = "belirtilmemis"
)

type SozlesmeDurumu string

const (
	SozlesmeDurumuYuklendi            SozlesmeDurumu = "YUKLENDI"
	SozlesmeDurumuIsleniyor           SozlesmeDurumu = "ISLENIYOR"
	SozlesmeDurumuIncelenmeyiBekliyor SozlesmeDurumu = "INCELENMEYI_BEKLIYOR"
	SozlesmeDurumuOnaylandi           SozlesmeDurumu = "ONAYLANDI"
	SozlesmeDurumuHata                SozlesmeDurumu = "HATA"
)

type BulguOnemi string

const (
	BulguOnemiKritik BulguOnemi = "KRITIK"
	BulguOnemiUyari  BulguOnemi = "UYARI"
	BulguOnemiBilgi  BulguOnemi = "BILGI"
)

type BulguKaynagi string

const (
	BulguKaynagiKural BulguKaynagi = "KURAL"
	BulguKaynagiModel BulguKaynagi = "MODEL"
)
