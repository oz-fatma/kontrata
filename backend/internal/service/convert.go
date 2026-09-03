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

func fromInput(g model.SozlesmeGirdi) repository.Contract {
	doc := repository.Contract{
		FileName:       g.DosyaAdi,
		RoomAllotments: []repository.RoomAllotment{},
		Prices:         []repository.Price{},
		StopSale:       []repository.StopSaleRange{},
	}
	if g.Durum != nil {
		doc.Status = string(*g.Durum)
	}
	if g.Meta != nil {
		doc.Meta = &repository.ContractMeta{
			HotelName:      g.Meta.OtelAdi,
			AgencyName:     g.Meta.AcenteAdi,
			ContractType:   enumPtr(g.Meta.SozlesmeTipi),
			Season:         enumPtr(g.Meta.Sezon),
			Currency:       g.Meta.ParaBirimi,
			ExchangeBasis:  enumPtr(g.Meta.KurEsasi),
			CompetentCourt: g.Meta.YetkiliMahkeme,
			SignatureDate:  g.Meta.ImzaTarihi,
		}
	}
	if g.Donem != nil {
		d := &repository.Period{Start: g.Donem.Baslangic, End: g.Donem.Bitis}
		for _, a := range g.Donem.AltDonemler {
			if a == nil {
				continue
			}
			d.SubPeriods = append(d.SubPeriods, repository.SubPeriod{Name: a.Ad, Start: a.Baslangic, End: a.Bitis})
		}
		doc.Period = d
	}
	for _, o := range g.OdaKontenjanlari {
		if o == nil {
			continue
		}
		doc.RoomAllotments = append(doc.RoomAllotments, repository.RoomAllotment{RoomType: o.OdaTipi, Quantity: o.Adet, Description: o.Aciklama})
	}
	for _, f := range g.Fiyatlar {
		if f == nil {
			continue
		}
		doc.Prices = append(doc.Prices, repository.Price{
			RoomType:      f.OdaTipi,
			Board:         enumPtr(f.Pansiyon),
			Amount:        f.Tutar,
			Unit:          string(f.Birim),
			SubPeriodName: f.AltDonemAd,
		})
	}
	if g.Release != nil {
		doc.Release = &repository.ReleaseRule{Days: g.Release.Gun, Scope: enumPtr(g.Release.Kapsam), SourcePhrase: g.Release.KaynakIfade}
	}
	for _, s := range g.StopSale {
		if s == nil {
			continue
		}
		doc.StopSale = append(doc.StopSale, repository.StopSaleRange{
			Start: s.Baslangic, End: s.Bitis, Scope: s.Kapsam,
			NotificationMethod: enumPtr(s.BildirimYontemi), SourcePhrase: s.KaynakIfade,
		})
	}
	for _, c := range g.CocukPolitikasi {
		if c == nil {
			continue
		}
		doc.ChildPolicies = append(doc.ChildPolicies, repository.ChildPolicy{
			AgeMin: c.YasMin, AgeMax: c.YasMax, DiscountPercent: c.IndirimYuzde, Free: c.Ucretsiz, Condition: c.Kosul,
		})
	}
	for _, i := range g.IptalKosullari {
		if i == nil {
			continue
		}
		doc.CancellationTerms = append(doc.CancellationTerms, repository.CancellationTerm{Scope: i.Kapsam, Days: i.Gun, CompensationNote: i.TazminatAciklama})
	}
	if g.NoShow != nil {
		doc.NoShow = &repository.NoShow{ResponsibleParty: g.NoShow.SorumluTaraf, CompensationNote: g.NoShow.TazminatAciklama}
	}
	if g.Overbooking != nil {
		doc.Overbooking = &repository.Overbooking{ResponsibleParty: g.Overbooking.SorumluTaraf, Description: g.Overbooking.Aciklama}
	}
	if g.Odeme != nil {
		doc.Payment = &repository.Payment{DaysAfterInvoice: g.Odeme.FaturaSonrasiGun, HasAdvance: g.Odeme.AvansVar, AdvanceNote: g.Odeme.AvansAciklama}
	}
	for _, c := range g.CikarimMeta {
		if c == nil {
			continue
		}
		doc.ExtractionMeta = append(doc.ExtractionMeta, repository.ExtractionMeta{FieldPath: c.AlanYolu, Confidence: c.Guven, SourcePage: c.KaynakSayfa, SourceClause: c.KaynakMadde})
	}
	return doc
}

func mapModelFindings(in []repository.Finding) []*model.Bulgu {
	out := make([]*model.Bulgu, 0, len(in))
	for i := range in {
		f := in[i]
		out = append(out, &model.Bulgu{
			Kod:      f.Code,
			Baslik:   f.Title,
			Aciklama: f.Description,
			Onem:     model.BulguOnemi(f.Severity),
			Kaynak:   model.BulguKaynagi(f.Source),
			AlanYolu: f.FieldPath,
		})
	}
	return out
}

func toModel(doc *repository.Contract) *model.Sozlesme {
	if doc == nil {
		return nil
	}
	out := &model.Sozlesme{
		ID:               doc.ID.Hex(),
		OlusturmaTarihi:  doc.CreatedAt,
		GuncellemeTarihi: doc.UpdatedAt,
		Durum:            model.SozlesmeDurumu(doc.Status),
		DosyaAdi:         doc.FileName,
		OdaKontenjanlari: []*model.OdaKontenjani{},
		Fiyatlar:         []*model.Fiyat{},
		StopSale:         []*model.StopSaleAraligi{},
		Bulgular:         []*model.Bulgu{},
	}
	if doc.Meta != nil {
		out.Meta = &model.SozlesmeMeta{
			OtelAdi:        doc.Meta.HotelName,
			AcenteAdi:      doc.Meta.AgencyName,
			SozlesmeTipi:   toEnum[model.SozlesmeTipi](doc.Meta.ContractType),
			Sezon:          toEnum[model.Sezon](doc.Meta.Season),
			ParaBirimi:     doc.Meta.Currency,
			KurEsasi:       toEnum[model.KurEsasi](doc.Meta.ExchangeBasis),
			YetkiliMahkeme: doc.Meta.CompetentCourt,
			ImzaTarihi:     doc.Meta.SignatureDate,
		}
	}
	if doc.Period != nil {
		d := &model.Donem{Baslangic: doc.Period.Start, Bitis: doc.Period.End}
		for i := range doc.Period.SubPeriods {
			a := doc.Period.SubPeriods[i]
			d.AltDonemler = append(d.AltDonemler, &model.AltDonem{Ad: a.Name, Baslangic: a.Start, Bitis: a.End})
		}
		out.Donem = d
	}
	out.Duzeltmeler = append([]string{}, doc.Repairs...)
	out.SemaHatalari = append([]string{}, doc.SchemaErrors...)
	out.IslemSuresi = doc.ProcessingSeconds
	out.DenetciSuresi = doc.AuditorSeconds
	out.Bulgular = mapModelFindings(doc.Findings)
	for i := range doc.RoomAllotments {
		o := doc.RoomAllotments[i]
		out.OdaKontenjanlari = append(out.OdaKontenjanlari, &model.OdaKontenjani{OdaTipi: o.RoomType, Adet: o.Quantity, Aciklama: o.Description})
	}
	for i := range doc.Prices {
		f := doc.Prices[i]
		out.Fiyatlar = append(out.Fiyatlar, &model.Fiyat{
			OdaTipi: f.RoomType, Pansiyon: toEnum[model.Pansiyon](f.Board),
			Tutar: f.Amount, Birim: model.FiyatBirimi(f.Unit), AltDonemAd: f.SubPeriodName,
		})
	}
	if doc.Release != nil {
		out.Release = &model.ReleaseKurali{Gun: doc.Release.Days, Kapsam: toEnum[model.ReleaseKapsami](doc.Release.Scope), KaynakIfade: doc.Release.SourcePhrase}
	}
	for i := range doc.StopSale {
		s := doc.StopSale[i]
		out.StopSale = append(out.StopSale, &model.StopSaleAraligi{
			Baslangic: s.Start, Bitis: s.End, Kapsam: s.Scope,
			BildirimYontemi: toEnum[model.BildirimYontemi](s.NotificationMethod), KaynakIfade: s.SourcePhrase,
		})
	}
	for i := range doc.ChildPolicies {
		c := doc.ChildPolicies[i]
		out.CocukPolitikasi = append(out.CocukPolitikasi, &model.CocukPolitikasi{
			YasMin: c.AgeMin, YasMax: c.AgeMax, IndirimYuzde: c.DiscountPercent, Ucretsiz: c.Free, Kosul: c.Condition,
		})
	}
	for i := range doc.CancellationTerms {
		k := doc.CancellationTerms[i]
		out.IptalKosullari = append(out.IptalKosullari, &model.IptalKosulu{Kapsam: k.Scope, Gun: k.Days, TazminatAciklama: k.CompensationNote})
	}
	if doc.NoShow != nil {
		out.NoShow = &model.NoShow{SorumluTaraf: doc.NoShow.ResponsibleParty, TazminatAciklama: doc.NoShow.CompensationNote}
	}
	if doc.Overbooking != nil {
		out.Overbooking = &model.Overbooking{SorumluTaraf: doc.Overbooking.ResponsibleParty, Aciklama: doc.Overbooking.Description}
	}
	if doc.Payment != nil {
		out.Odeme = &model.Odeme{FaturaSonrasiGun: doc.Payment.DaysAfterInvoice, AvansVar: doc.Payment.HasAdvance, AvansAciklama: doc.Payment.AdvanceNote}
	}
	for i := range doc.ExtractionMeta {
		c := doc.ExtractionMeta[i]
		out.CikarimMeta = append(out.CikarimMeta, &model.CikarimMeta{AlanYolu: c.FieldPath, Guven: c.Confidence, KaynakSayfa: c.SourcePage, KaynakMadde: c.SourceClause})
	}
	return out
}
