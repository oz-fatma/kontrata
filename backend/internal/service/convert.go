package service

import (
	"github.com/oz-fatma/kontrata/backend/graph/model"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

func enumPtr[T ~string](v *T) *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}

func toEnum[T ~string](v *string) *T {
	if v == nil {
		return nil
	}
	t := T(*v)
	return &t
}

func fromGirdi(g model.SozlesmeGirdi) repository.Sozlesme {
	doc := repository.Sozlesme{
		DosyaAdi:         g.DosyaAdi,
		OdaKontenjanlari: []repository.OdaKontenjani{},
		Fiyatlar:         []repository.Fiyat{},
		StopSale:         []repository.StopSaleAraligi{},
	}
	if g.Durum != nil {
		doc.Durum = string(*g.Durum)
	}
	if g.Meta != nil {
		doc.Meta = &repository.SozlesmeMeta{
			OtelAdi:        g.Meta.OtelAdi,
			AcenteAdi:      g.Meta.AcenteAdi,
			SozlesmeTipi:   enumPtr(g.Meta.SozlesmeTipi),
			Sezon:          enumPtr(g.Meta.Sezon),
			ParaBirimi:     g.Meta.ParaBirimi,
			KurEsasi:       enumPtr(g.Meta.KurEsasi),
			YetkiliMahkeme: g.Meta.YetkiliMahkeme,
			ImzaTarihi:     g.Meta.ImzaTarihi,
		}
	}
	if g.Donem != nil {
		d := &repository.Donem{Baslangic: g.Donem.Baslangic, Bitis: g.Donem.Bitis}
		for _, a := range g.Donem.AltDonemler {
			if a == nil {
				continue
			}
			d.AltDonemler = append(d.AltDonemler, repository.AltDonem{Ad: a.Ad, Baslangic: a.Baslangic, Bitis: a.Bitis})
		}
		doc.Donem = d
	}
	for _, o := range g.OdaKontenjanlari {
		if o == nil {
			continue
		}
		doc.OdaKontenjanlari = append(doc.OdaKontenjanlari, repository.OdaKontenjani{OdaTipi: o.OdaTipi, Adet: o.Adet, Aciklama: o.Aciklama})
	}
	for _, f := range g.Fiyatlar {
		if f == nil {
			continue
		}
		doc.Fiyatlar = append(doc.Fiyatlar, repository.Fiyat{
			OdaTipi:    f.OdaTipi,
			Pansiyon:   enumPtr(f.Pansiyon),
			Tutar:      f.Tutar,
			Birim:      string(f.Birim),
			AltDonemAd: f.AltDonemAd,
		})
	}
	if g.Release != nil {
		doc.Release = &repository.ReleaseKurali{Gun: g.Release.Gun, Kapsam: enumPtr(g.Release.Kapsam), KaynakIfade: g.Release.KaynakIfade}
	}
	for _, s := range g.StopSale {
		if s == nil {
			continue
		}
		doc.StopSale = append(doc.StopSale, repository.StopSaleAraligi{
			Baslangic: s.Baslangic, Bitis: s.Bitis, Kapsam: s.Kapsam,
			BildirimYontemi: enumPtr(s.BildirimYontemi), KaynakIfade: s.KaynakIfade,
		})
	}
	for _, c := range g.CocukPolitikasi {
		if c == nil {
			continue
		}
		doc.CocukPolitikasi = append(doc.CocukPolitikasi, repository.CocukPolitikasi{
			YasMin: c.YasMin, YasMax: c.YasMax, IndirimYuzde: c.IndirimYuzde, Ucretsiz: c.Ucretsiz, Kosul: c.Kosul,
		})
	}
	for _, i := range g.IptalKosullari {
		if i == nil {
			continue
		}
		doc.IptalKosullari = append(doc.IptalKosullari, repository.IptalKosulu{Kapsam: i.Kapsam, Gun: i.Gun, TazminatAciklama: i.TazminatAciklama})
	}
	if g.NoShow != nil {
		doc.NoShow = &repository.NoShow{SorumluTaraf: g.NoShow.SorumluTaraf, TazminatAciklama: g.NoShow.TazminatAciklama}
	}
	if g.Overbooking != nil {
		doc.Overbooking = &repository.Overbooking{SorumluTaraf: g.Overbooking.SorumluTaraf, Aciklama: g.Overbooking.Aciklama}
	}
	if g.Odeme != nil {
		doc.Odeme = &repository.Odeme{FaturaSonrasiGun: g.Odeme.FaturaSonrasiGun, AvansVar: g.Odeme.AvansVar, AvansAciklama: g.Odeme.AvansAciklama}
	}
	for _, c := range g.CikarimMeta {
		if c == nil {
			continue
		}
		doc.CikarimMeta = append(doc.CikarimMeta, repository.CikarimMeta{AlanYolu: c.AlanYolu, Guven: c.Guven, KaynakSayfa: c.KaynakSayfa, KaynakMadde: c.KaynakMadde})
	}
	return doc
}

func toModel(doc *repository.Sozlesme) *model.Sozlesme {
	if doc == nil {
		return nil
	}
	out := &model.Sozlesme{
		ID:               doc.ID.Hex(),
		OlusturmaTarihi:  doc.OlusturmaTarihi,
		GuncellemeTarihi: doc.GuncellemeTarihi,
		Durum:            model.SozlesmeDurumu(doc.Durum),
		DosyaAdi:         doc.DosyaAdi,
		OdaKontenjanlari: []*model.OdaKontenjani{},
		Fiyatlar:         []*model.Fiyat{},
		StopSale:         []*model.StopSaleAraligi{},
	}
	if doc.Meta != nil {
		out.Meta = &model.SozlesmeMeta{
			OtelAdi:        doc.Meta.OtelAdi,
			AcenteAdi:      doc.Meta.AcenteAdi,
			SozlesmeTipi:   toEnum[model.SozlesmeTipi](doc.Meta.SozlesmeTipi),
			Sezon:          toEnum[model.Sezon](doc.Meta.Sezon),
			ParaBirimi:     doc.Meta.ParaBirimi,
			KurEsasi:       toEnum[model.KurEsasi](doc.Meta.KurEsasi),
			YetkiliMahkeme: doc.Meta.YetkiliMahkeme,
			ImzaTarihi:     doc.Meta.ImzaTarihi,
		}
	}
	if doc.Donem != nil {
		d := &model.Donem{Baslangic: doc.Donem.Baslangic, Bitis: doc.Donem.Bitis}
		for i := range doc.Donem.AltDonemler {
			a := doc.Donem.AltDonemler[i]
			d.AltDonemler = append(d.AltDonemler, &model.AltDonem{Ad: a.Ad, Baslangic: a.Baslangic, Bitis: a.Bitis})
		}
		out.Donem = d
	}
	for i := range doc.OdaKontenjanlari {
		o := doc.OdaKontenjanlari[i]
		out.OdaKontenjanlari = append(out.OdaKontenjanlari, &model.OdaKontenjani{OdaTipi: o.OdaTipi, Adet: o.Adet, Aciklama: o.Aciklama})
	}
	for i := range doc.Fiyatlar {
		f := doc.Fiyatlar[i]
		out.Fiyatlar = append(out.Fiyatlar, &model.Fiyat{
			OdaTipi: f.OdaTipi, Pansiyon: toEnum[model.Pansiyon](f.Pansiyon),
			Tutar: f.Tutar, Birim: model.FiyatBirimi(f.Birim), AltDonemAd: f.AltDonemAd,
		})
	}
	if doc.Release != nil {
		out.Release = &model.ReleaseKurali{Gun: doc.Release.Gun, Kapsam: toEnum[model.ReleaseKapsami](doc.Release.Kapsam), KaynakIfade: doc.Release.KaynakIfade}
	}
	for i := range doc.StopSale {
		s := doc.StopSale[i]
		out.StopSale = append(out.StopSale, &model.StopSaleAraligi{
			Baslangic: s.Baslangic, Bitis: s.Bitis, Kapsam: s.Kapsam,
			BildirimYontemi: toEnum[model.BildirimYontemi](s.BildirimYontemi), KaynakIfade: s.KaynakIfade,
		})
	}
	for i := range doc.CocukPolitikasi {
		c := doc.CocukPolitikasi[i]
		out.CocukPolitikasi = append(out.CocukPolitikasi, &model.CocukPolitikasi{
			YasMin: c.YasMin, YasMax: c.YasMax, IndirimYuzde: c.IndirimYuzde, Ucretsiz: c.Ucretsiz, Kosul: c.Kosul,
		})
	}
	for i := range doc.IptalKosullari {
		k := doc.IptalKosullari[i]
		out.IptalKosullari = append(out.IptalKosullari, &model.IptalKosulu{Kapsam: k.Kapsam, Gun: k.Gun, TazminatAciklama: k.TazminatAciklama})
	}
	if doc.NoShow != nil {
		out.NoShow = &model.NoShow{SorumluTaraf: doc.NoShow.SorumluTaraf, TazminatAciklama: doc.NoShow.TazminatAciklama}
	}
	if doc.Overbooking != nil {
		out.Overbooking = &model.Overbooking{SorumluTaraf: doc.Overbooking.SorumluTaraf, Aciklama: doc.Overbooking.Aciklama}
	}
	if doc.Odeme != nil {
		out.Odeme = &model.Odeme{FaturaSonrasiGun: doc.Odeme.FaturaSonrasiGun, AvansVar: doc.Odeme.AvansVar, AvansAciklama: doc.Odeme.AvansAciklama}
	}
	for i := range doc.CikarimMeta {
		c := doc.CikarimMeta[i]
		out.CikarimMeta = append(out.CikarimMeta, &model.CikarimMeta{AlanYolu: c.AlanYolu, Guven: c.Guven, KaynakSayfa: c.KaynakSayfa, KaynakMadde: c.KaynakMadde})
	}
	return out
}
