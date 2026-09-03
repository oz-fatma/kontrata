package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/oz-fatma/kontrata/backend/graph/model"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

var graphqlPath = map[string]string{
	"meta.otelAdi":         "meta.otelAdi",
	"meta.otel_adi":        "meta.otelAdi",
	"meta.acenteAdi":       "meta.acenteAdi",
	"meta.acente_adi":      "meta.acenteAdi",
	"meta.sozlesmeTipi":    "meta.sozlesmeTipi",
	"meta.sozlesme_tipi":   "meta.sozlesmeTipi",
	"meta.sezon":           "meta.sezon",
	"meta.paraBirimi":      "meta.paraBirimi",
	"meta.para_birimi":     "meta.paraBirimi",
	"meta.kurEsasi":        "meta.kurEsasi",
	"meta.kur_esasi":       "meta.kurEsasi",
	"meta.yetkiliMahkeme":  "meta.yetkiliMahkeme",
	"meta.yetkili_mahkeme": "meta.yetkiliMahkeme",
	"meta.imzaTarihi":      "meta.imzaTarihi",
	"meta.imza_tarihi":     "meta.imzaTarihi",
	"donem":                "donem",
	"donem.baslangic":      "donem.baslangic",
	"donem.bitis":          "donem.bitis",
	"odaKontenjanlari":     "odaKontenjanlari",
	"oda_kontenjanlari":    "odaKontenjanlari",
	"fiyatlar":             "fiyatlar",
	"release":              "release",
	"stopSale":             "stopSale",
	"stop_sale":            "stopSale",
}

func applyFieldValue(doc *repository.Contract, path string, value any) (string, error) {
	if doc == nil {
		return "", ErrUnknownField
	}
	key, ok := graphqlPath[strings.TrimSpace(path)]
	if !ok {
		return "", ErrUnknownField
	}
	switch key {
	case "meta.otelAdi":
		s, err := requireString(value)
		if err != nil {
			return "", err
		}
		ensureMeta(doc).HotelName = strOrNil(s)
	case "meta.acenteAdi":
		s, err := requireString(value)
		if err != nil {
			return "", err
		}
		ensureMeta(doc).AgencyName = strOrNil(s)
	case "meta.sozlesmeTipi":
		s, err := parseMappedEnum(value, sozlesmeTipiMap)
		if err != nil {
			return "", err
		}
		ensureMeta(doc).ContractType = strOrNil(s)
	case "meta.sezon":
		s, err := parseMappedEnum(value, sezonMap)
		if err != nil {
			return "", err
		}
		ensureMeta(doc).Season = strOrNil(s)
	case "meta.paraBirimi":
		s, err := requireString(value)
		if err != nil {
			return "", err
		}
		ensureMeta(doc).Currency = strOrNil(s)
	case "meta.kurEsasi":
		s, err := parseMappedEnum(value, kurEsasiMap)
		if err != nil {
			return "", err
		}
		ensureMeta(doc).ExchangeBasis = strOrNil(s)
	case "meta.yetkiliMahkeme":
		s, err := requireString(value)
		if err != nil {
			return "", err
		}
		ensureMeta(doc).CompetentCourt = strOrNil(s)
	case "meta.imzaTarihi":
		s, err := requireString(value)
		if err != nil {
			return "", err
		}
		ensureMeta(doc).SignatureDate = strOrNil(s)
	case "donem":
		p, err := parsePeriodValue(value)
		if err != nil {
			return "", err
		}
		if doc.Period != nil {
			p.SubPeriods = doc.Period.SubPeriods
		}
		doc.Period = p
	case "donem.baslangic":
		s, err := requireString(value)
		if err != nil {
			return "", err
		}
		ensurePeriod(doc).Start = strOrNil(s)
	case "donem.bitis":
		s, err := requireString(value)
		if err != nil {
			return "", err
		}
		ensurePeriod(doc).End = strOrNil(s)
	case "odaKontenjanlari":
		rooms, err := parseRoomsValue(value)
		if err != nil {
			return "", err
		}
		doc.RoomAllotments = rooms
	case "fiyatlar":
		prices, err := parsePricesValue(value)
		if err != nil {
			return "", err
		}
		doc.Prices = prices
	case "release":
		rel, err := parseReleaseValue(value)
		if err != nil {
			return "", err
		}
		doc.Release = rel
	case "stopSale":
		rows, err := parseStopSaleValue(value)
		if err != nil {
			return "", err
		}
		doc.StopSale = rows
	default:
		return "", ErrUnknownField
	}
	return key, nil
}

func markFieldManuallyFixed(doc *repository.Contract, path string) {
	if doc == nil || path == "" {
		return
	}
	one := 1.0
	for i := range doc.ExtractionMeta {
		if doc.ExtractionMeta[i].FieldPath == path {
			doc.ExtractionMeta[i].Confidence = &one
			doc.ExtractionMeta[i].ManuallyFixed = true
			return
		}
	}
	doc.ExtractionMeta = append(doc.ExtractionMeta, repository.ExtractionMeta{
		FieldPath:     path,
		Confidence:    &one,
		ManuallyFixed: true,
	})
}

func ensureMeta(doc *repository.Contract) *repository.ContractMeta {
	if doc.Meta == nil {
		doc.Meta = &repository.ContractMeta{}
	}
	return doc.Meta
}

func ensurePeriod(doc *repository.Contract) *repository.Period {
	if doc.Period == nil {
		doc.Period = &repository.Period{}
	}
	return doc.Period
}

func strOrNil(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func requireString(v any) (string, error) {
	if v == nil {
		return "", nil
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x), nil
	case fmt.Stringer:
		return strings.TrimSpace(x.String()), nil
	default:
		return "", ErrInvalidFieldValue
	}
}

func parseMappedEnum(v any, m map[string]string) (string, error) {
	s, err := requireString(v)
	if err != nil {
		return "", err
	}
	if s == "" {
		return "", nil
	}
	if mapped, ok := m[strings.ToUpper(strings.TrimSpace(s))]; ok {
		return mapped, nil
	}
	key := strings.ToLower(strings.TrimSpace(s))
	if mapped, ok := m[key]; ok {
		return mapped, nil
	}
	return "", ErrInvalidFieldValue
}

func parsePeriodValue(v any) (*repository.Period, error) {
	if m := asMap(v); m != nil {
		return &repository.Period{
			Start: strPtr(firstKey(m, "baslangic", "start")),
			End:   strPtr(firstKey(m, "bitis", "end")),
		}, nil
	}
	s, err := requireString(v)
	if err != nil {
		return nil, err
	}
	start, end, ok := splitPeriod(s)
	if !ok {
		return nil, ErrInvalidFieldValue
	}
	return &repository.Period{Start: strOrNil(start), End: strOrNil(end)}, nil
}

func splitPeriod(s string) (string, string, bool) {
	s = strings.TrimSpace(s)
	for _, sep := range []string{" – ", " — ", " - ", "–", "—"} {
		if a, b, ok := strings.Cut(s, sep); ok {
			return strings.TrimSpace(a), strings.TrimSpace(b), true
		}
	}
	return "", "", false
}

func parseRoomsValue(v any) ([]repository.RoomAllotment, error) {
	if arr := anySlice(v); arr != nil {
		normalized := make([]any, 0, len(arr))
		for _, item := range arr {
			m := asMap(item)
			if m == nil {
				continue
			}
			normalized = append(normalized, map[string]any{
				"oda_tipi": firstKey(m, "oda_tipi", "odaTipi"),
				"adet":     firstKey(m, "adet"),
				"aciklama": firstKey(m, "aciklama"),
			})
		}
		out := mapRooms(normalized)
		if out == nil {
			out = []repository.RoomAllotment{}
		}
		return out, nil
	}
	s, err := requireString(v)
	if err != nil {
		return nil, err
	}
	out := []repository.RoomAllotment{}
	for _, line := range nonEmptyLines(s) {
		tip, adet, aciklama, ok := parseRoomLine(line)
		if !ok {
			return nil, ErrInvalidFieldValue
		}
		out = append(out, repository.RoomAllotment{RoomType: tip, Quantity: adet, Description: strOrNil(aciklama)})
	}
	return out, nil
}

func parseRoomLine(line string) (string, int32, string, bool) {
	tip, rest, ok := strings.Cut(line, ":")
	if !ok {
		return "", 0, "", false
	}
	tip = strings.TrimSpace(tip)
	rest = strings.TrimSpace(rest)
	aciklama := ""
	if i := strings.Index(rest, " ("); i >= 0 && strings.HasSuffix(rest, ")") {
		aciklama = strings.TrimSuffix(rest[i+2:], ")")
		rest = strings.TrimSpace(rest[:i])
	}
	n, err := strconv.Atoi(rest)
	if err != nil || tip == "" {
		return "", 0, "", false
	}
	return tip, int32(n), aciklama, true
}

func parsePricesValue(v any) ([]repository.Price, error) {
	if arr := anySlice(v); arr != nil {
		normalized := make([]any, 0, len(arr))
		for _, item := range arr {
			m := asMap(item)
			if m == nil {
				continue
			}
			birim := firstKey(m, "birim")
			if s, ok := birim.(string); ok {
				if mapped, err := parseMappedEnum(s, fiyatBirimiMap); err == nil {
					birim = mapped
				}
			}
			pansiyon := firstKey(m, "pansiyon")
			if s, ok := pansiyon.(string); ok && s != "" {
				if mapped, err := parseMappedEnum(s, pansiyonMap); err == nil {
					pansiyon = mapped
				}
			}
			normalized = append(normalized, map[string]any{
				"oda_tipi":     firstKey(m, "oda_tipi", "odaTipi"),
				"pansiyon":     pansiyon,
				"tutar":        firstKey(m, "tutar"),
				"birim":        birim,
				"alt_donem_ad": firstKey(m, "alt_donem_ad", "altDonemAd"),
			})
		}
		out := mapPrices(normalized)
		if out == nil {
			out = []repository.Price{}
		}
		return out, nil
	}
	s, err := requireString(v)
	if err != nil {
		return nil, err
	}
	out := []repository.Price{}
	for _, line := range nonEmptyLines(s) {
		p, ok := parsePriceLine(line)
		if !ok {
			return nil, ErrInvalidFieldValue
		}
		out = append(out, p)
	}
	return out, nil
}

func parsePriceLine(line string) (repository.Price, bool) {
	parts := strings.Split(line, " · ")
	if len(parts) < 3 {
		return repository.Price{}, false
	}
	tip := strings.TrimSpace(parts[0])
	board := strings.TrimSpace(parts[1])
	rest := strings.TrimSpace(strings.Join(parts[2:], " · "))
	amountPart, unitPart, ok := strings.Cut(rest, " (")
	if !ok || !strings.HasSuffix(unitPart, ")") {
		return repository.Price{}, false
	}
	unitPart = strings.TrimSuffix(unitPart, ")")
	amount, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(amountPart), ",", "."), 64)
	if err != nil || tip == "" {
		return repository.Price{}, false
	}
	unit, err := parseMappedEnum(unitPart, fiyatBirimiMap)
	if err != nil {
		return repository.Price{}, false
	}
	p := repository.Price{RoomType: tip, Amount: amount, Unit: unit}
	if board != "" && board != "—" {
		if mapped, mapErr := parseMappedEnum(board, pansiyonMap); mapErr == nil {
			p.Board = strOrNil(mapped)
		} else {
			p.Board = strOrNil(board)
		}
	}
	return p, true
}

func parseReleaseValue(v any) (*repository.ReleaseRule, error) {
	if m := asMap(v); m != nil {
		return mapRelease(m), nil
	}
	s, err := requireString(v)
	if err != nil {
		return nil, err
	}
	gunPart, rest, ok := strings.Cut(strings.TrimSpace(s), " gün")
	if !ok {
		return nil, ErrInvalidFieldValue
	}
	n, convErr := strconv.Atoi(strings.TrimSpace(gunPart))
	if convErr != nil {
		return nil, ErrInvalidFieldValue
	}
	rel := &repository.ReleaseRule{Days: int32(n)}
	rest = strings.TrimLeft(strings.TrimSpace(rest), "· ")
	if rest != "" && rest != "—" {
		if mapped, mapErr := parseMappedEnum(rest, releaseKapsamMap); mapErr == nil {
			rel.Scope = strOrNil(mapped)
		}
	}
	return rel, nil
}

func parseStopSaleValue(v any) ([]repository.StopSaleRange, error) {
	if arr := anySlice(v); arr != nil {
		out := mapStopSale(arr)
		if out == nil {
			out = []repository.StopSaleRange{}
		}
		return out, nil
	}
	s, err := requireString(v)
	if err != nil {
		return nil, err
	}
	out := []repository.StopSaleRange{}
	for _, line := range nonEmptyLines(s) {
		period, rest, _ := strings.Cut(line, " · ")
		start, end, ok := splitPeriod(strings.TrimSpace(period))
		if !ok {
			return nil, ErrInvalidFieldValue
		}
		out = append(out, repository.StopSaleRange{
			Start: strOrNil(start),
			End:   strOrNil(end),
			Scope: strOrNil(strings.TrimSpace(rest)),
		})
	}
	return out, nil
}

func nonEmptyLines(s string) []string {
	raw := strings.Split(s, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func anySlice(v any) []any {
	switch x := v.(type) {
	case []any:
		return x
	case []map[string]any:
		out := make([]any, 0, len(x))
		for i := range x {
			out = append(out, x[i])
		}
		return out
	default:
		return nil
	}
}

func firstKey(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}

var (
	sozlesmeTipiMap = mergeEnumMaps(
		enumAliases(string(model.SozlesmeTipiTamamenGarantili), "TAMAMEN_GARANTILI", "tamamen garantili"),
		enumAliases(string(model.SozlesmeTipiKismenGarantili), "KISMEN_GARANTILI", "kısmen garantili"),
		enumAliases(string(model.SozlesmeTipiGarantisiz), "GARANTISIZ", "garantisiz"),
		enumAliases(string(model.SozlesmeTipiIstegeBagli), "ISTEGE_BAGLI", "isteğe bağlı", "istege bagli"),
		enumAliases(string(model.SozlesmeTipiSerbestSatis), "SERBEST_SATIS", "serbest satış", "serbest satis"),
		enumAliases(string(model.SozlesmeTipiBlokRezervasyon), "BLOK_REZERVASYON", "blok rezervasyon"),
		enumAliases(string(model.SozlesmeTipiBlokSatinAlma), "BLOK_SATIN_ALMA", "blok satın alma", "blok satin alma"),
		enumAliases(string(model.SozlesmeTipiBelirtilmemis), "BELIRTILMEMIS", "belirtilmemiş", "belirtilmemis"),
	)
	sezonMap = mergeEnumMaps(
		enumAliases(string(model.SezonYaz), "YAZ", "yaz"),
		enumAliases(string(model.SezonKis), "KIS", "kış", "kis"),
		enumAliases(string(model.SezonYillik), "YILLIK", "yıllık", "yillik"),
		enumAliases(string(model.SezonBelirtilmemis), "BELIRTILMEMIS", "belirtilmemiş", "belirtilmemis"),
	)
	kurEsasiMap = mergeEnumMaps(
		enumAliases(string(model.KurEsasiGirisGunuTcmb), "GIRIS_GUNU_TCMB", "giriş günü tcmb", "giris gunu tcmb"),
		enumAliases(string(model.KurEsasiCikisGunuTcmb), "CIKIS_GUNU_TCMB", "çıkış günü tcmb", "cikis gunu tcmb"),
		enumAliases(string(model.KurEsasiSabitKur), "SABIT_KUR", "sabit kur"),
		enumAliases(string(model.KurEsasiBelirtilmemis), "BELIRTILMEMIS", "belirtilmemiş", "belirtilmemis"),
	)
	fiyatBirimiMap = mergeEnumMaps(
		enumAliases(string(model.FiyatBirimiOdaGecelik), "ODA_GECELIK", "oda gecelik"),
		enumAliases(string(model.FiyatBirimiKisiGecelik), "KISI_GECELIK", "kişi gecelik", "kisi gecelik"),
	)
	pansiyonMap = mergeEnumMaps(
		enumAliases(string(model.PansiyonRo), "RO"),
		enumAliases(string(model.PansiyonBb), "BB"),
		enumAliases(string(model.PansiyonHb), "HB"),
		enumAliases(string(model.PansiyonFb), "FB"),
		enumAliases(string(model.PansiyonAi), "AI"),
		enumAliases(string(model.PansiyonBelirtilmemis), "BELIRTILMEMIS", "belirtilmemiş", "belirtilmemis"),
	)
	releaseKapsamMap = mergeEnumMaps(
		enumAliases(string(model.ReleaseKapsamiIsimListesi), "ISIM_LISTESI", "isim listesi"),
		enumAliases(string(model.ReleaseKapsamiKontenjanIadesi), "KONTENJAN_IADESI", "kontenjan iadesi"),
		enumAliases(string(model.ReleaseKapsamiHerIkisi), "HER_IKISI", "her ikisi"),
		enumAliases(string(model.ReleaseKapsamiBelirtilmemis), "BELIRTILMEMIS", "belirtilmemiş", "belirtilmemis"),
	)
)

func enumAliases(store string, aliases ...string) map[string]string {
	m := map[string]string{
		strings.ToUpper(store): store,
		strings.ToLower(store): store,
	}
	for _, a := range aliases {
		m[strings.ToUpper(a)] = store
		m[strings.ToLower(a)] = store
	}
	return m
}

func mergeEnumMaps(parts ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, p := range parts {
		for k, v := range p {
			out[k] = v
		}
	}
	return out
}
