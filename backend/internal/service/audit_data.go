package service

import (
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

func auditDataFromContract(doc *repository.Contract) map[string]any {
	if doc == nil {
		return map[string]any{}
	}
	data := map[string]any{}
	if doc.Meta != nil {
		data["meta"] = map[string]any{
			"otel_adi":        ptrString(doc.Meta.HotelName),
			"acente_adi":      ptrString(doc.Meta.AgencyName),
			"sozlesme_tipi":   ptrString(doc.Meta.ContractType),
			"sezon":           ptrString(doc.Meta.Season),
			"para_birimi":     ptrString(doc.Meta.Currency),
			"kur_esasi":       ptrString(doc.Meta.ExchangeBasis),
			"yetkili_mahkeme": ptrString(doc.Meta.CompetentCourt),
			"imza_tarihi":     ptrString(doc.Meta.SignatureDate),
		}
	}
	if doc.Period != nil {
		donem := map[string]any{
			"baslangic": ptrString(doc.Period.Start),
			"bitis":     ptrString(doc.Period.End),
		}
		if len(doc.Period.SubPeriods) > 0 {
			alt := make([]any, 0, len(doc.Period.SubPeriods))
			for _, s := range doc.Period.SubPeriods {
				alt = append(alt, map[string]any{"ad": s.Name, "baslangic": s.Start, "bitis": s.End})
			}
			donem["alt_donemler"] = alt
		}
		data["donem"] = donem
	}
	rooms := make([]any, 0, len(doc.RoomAllotments))
	for _, o := range doc.RoomAllotments {
		rooms = append(rooms, map[string]any{
			"oda_tipi": o.RoomType,
			"adet":     o.Quantity,
			"aciklama": ptrString(o.Description),
		})
	}
	data["oda_kontenjanlari"] = rooms
	prices := make([]any, 0, len(doc.Prices))
	for _, f := range doc.Prices {
		prices = append(prices, map[string]any{
			"oda_tipi":     f.RoomType,
			"pansiyon":     ptrString(f.Board),
			"tutar":        f.Amount,
			"birim":        f.Unit,
			"alt_donem_ad": ptrString(f.SubPeriodName),
		})
	}
	data["fiyatlar"] = prices
	if doc.Release != nil {
		data["release"] = map[string]any{
			"gun":          doc.Release.Days,
			"kapsam":       ptrString(doc.Release.Scope),
			"kaynak_ifade": ptrString(doc.Release.SourcePhrase),
		}
	}
	stops := make([]any, 0, len(doc.StopSale))
	for _, s := range doc.StopSale {
		stops = append(stops, map[string]any{
			"baslangic":        ptrString(s.Start),
			"bitis":            ptrString(s.End),
			"kapsam":           ptrString(s.Scope),
			"bildirim_yontemi": ptrString(s.NotificationMethod),
			"kaynak_ifade":     ptrString(s.SourcePhrase),
		})
	}
	data["stop_sale"] = stops
	data["cikarim_meta"] = extractionMetaMap(doc.ExtractionMeta)
	return data
}

func extractionMetaMap(in []repository.ExtractionMeta) map[string]any {
	out := map[string]any{}
	for _, m := range in {
		schema := schemaPathOf(m.FieldPath)
		entry := map[string]any{}
		if m.Confidence != nil {
			entry["guven"] = *m.Confidence
		}
		out[schema] = entry
	}
	return out
}

func schemaPathOf(graphqlPath string) string {
	switch graphqlPath {
	case "odaKontenjanlari":
		return "oda_kontenjanlari"
	case "stopSale":
		return "stop_sale"
	default:
		return graphqlPath
	}
}

func ptrString(v *string) any {
	if v == nil {
		return ""
	}
	return *v
}
