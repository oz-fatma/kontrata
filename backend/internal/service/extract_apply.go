package service

import (
	"fmt"
	"log"
	"strings"

	"github.com/oz-fatma/kontrata/backend/internal/agent"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

// applyExtract overlays Okuyucu çıktısını mevcut kaydın üzerine yazar.
// Kimlik, dosya ve zaman damgaları korunur.
func applyExtract(doc *repository.Contract, res *agent.ExtractResult) {
	if doc == nil || res == nil {
		return
	}
	data := res.Data
	if data == nil {
		data = map[string]any{}
	}
	doc.Meta = mapMeta(asMap(data["meta"]))
	doc.Period = mapPeriod(asMap(data["donem"]))
	doc.RoomAllotments = mapRooms(asSlice(data["oda_kontenjanlari"]))
	doc.Prices = mapPrices(asSlice(data["fiyatlar"]))
	doc.Release = mapRelease(asMap(data["release"]))
	doc.StopSale = mapStopSale(asSlice(data["stop_sale"]))
	if op := asMap(data["opsiyonel"]); op != nil {
		doc.ChildPolicies = mapChildren(asSlice(op["cocuk_politikasi"]))
		doc.CancellationTerms = mapCancel(asSlice(op["iptal_kosullari"]))
		doc.NoShow = mapNoShow(asMap(op["no_show"]))
		doc.Overbooking = mapOverbooking(asMap(op["overbooking"]))
		doc.Payment = mapPayment(asMap(op["odeme"]))
	}
	doc.ExtractionMeta = mapExtractMeta(res.Meta)
	doc.Repairs = append([]string{}, res.Repairs...)
	doc.SchemaErrors = append([]string{}, res.SchemaErrors...)
	sec := res.Duration.Seconds()
	doc.ProcessingSeconds = &sec
}

func applyAudit(doc *repository.Contract, res *agent.AuditResult) {
	if doc == nil || res == nil {
		return
	}
	doc.Findings = mapFindings(res.Findings)
	total := res.RuleDuration + res.ModelDuration
	sec := int32(total.Seconds())
	if total > 0 && sec == 0 {
		sec = 1
	}
	doc.AuditorSeconds = &sec
}

func applyAuditOutcome(doc *repository.Contract, res *agent.AuditResult, err error) {
	if doc == nil {
		return
	}
	if err != nil {
		log.Printf("denetci atlandi: %v", err)
		return
	}
	applyAudit(doc, res)
}

func mapFindings(in []agent.Finding) []repository.Finding {
	out := make([]repository.Finding, 0, len(in))
	for _, f := range in {
		item := repository.Finding{
			Code:        f.Code,
			Title:       f.Title,
			Description: f.Description,
			Severity:    f.Severity,
			Source:      f.Source,
		}
		if p := strings.TrimSpace(f.FieldPath); p != "" {
			item.FieldPath = &p
		}
		out = append(out, item)
	}
	return out
}

func mapExtractMeta(in []agent.FieldMeta) []repository.ExtractionMeta {
	out := make([]repository.ExtractionMeta, 0, len(in))
	for _, m := range in {
		conf := m.Confidence
		out = append(out, repository.ExtractionMeta{
			FieldPath:  m.FieldPath,
			Confidence: &conf,
			SourcePage: m.SourcePage,
		})
	}
	return out
}

func mapMeta(m map[string]any) *repository.ContractMeta {
	if m == nil {
		return nil
	}
	return &repository.ContractMeta{
		HotelName:      strPtr(m["otel_adi"]),
		AgencyName:     strPtr(m["acente_adi"]),
		ContractType:   strPtr(m["sozlesme_tipi"]),
		Season:         strPtr(m["sezon"]),
		Currency:       strPtr(m["para_birimi"]),
		ExchangeBasis:  strPtr(m["kur_esasi"]),
		CompetentCourt: strPtr(m["yetkili_mahkeme"]),
		SignatureDate:  strPtr(m["imza_tarihi"]),
	}
}

func mapPeriod(m map[string]any) *repository.Period {
	if m == nil {
		return nil
	}
	p := &repository.Period{Start: strPtr(m["baslangic"]), End: strPtr(m["bitis"])}
	for _, item := range asSlice(m["alt_donemler"]) {
		am := asMap(item)
		if am == nil {
			continue
		}
		ad, ok1 := asString(am["ad"])
		b, ok2 := asString(am["baslangic"])
		e, ok3 := asString(am["bitis"])
		if !ok1 || !ok2 || !ok3 || ad == "" {
			continue
		}
		p.SubPeriods = append(p.SubPeriods, repository.SubPeriod{Name: ad, Start: b, End: e})
	}
	return p
}

func mapRooms(arr []any) []repository.RoomAllotment {
	out := []repository.RoomAllotment{}
	for _, item := range arr {
		m := asMap(item)
		if m == nil {
			continue
		}
		tip, ok := asString(m["oda_tipi"])
		adet, ok2 := asInt32(m["adet"])
		if !ok || !ok2 || tip == "" {
			continue
		}
		out = append(out, repository.RoomAllotment{RoomType: tip, Quantity: adet, Description: strPtr(m["aciklama"])})
	}
	return out
}

func mapPrices(arr []any) []repository.Price {
	out := []repository.Price{}
	for _, item := range arr {
		m := asMap(item)
		if m == nil {
			continue
		}
		tip, ok := asString(m["oda_tipi"])
		tutar, ok2 := asFloat(m["tutar"])
		birim, ok3 := asString(m["birim"])
		if !ok || !ok2 || !ok3 || tip == "" || birim == "" {
			continue
		}
		out = append(out, repository.Price{
			RoomType:      tip,
			Board:         strPtr(m["pansiyon"]),
			Amount:        tutar,
			Unit:          birim,
			SubPeriodName: strPtr(m["alt_donem_ad"]),
		})
	}
	return out
}

func mapRelease(m map[string]any) *repository.ReleaseRule {
	if m == nil {
		return nil
	}
	gun, ok := asInt32(m["gun"])
	if !ok {
		return nil
	}
	return &repository.ReleaseRule{Days: gun, Scope: strPtr(m["kapsam"]), SourcePhrase: strPtr(m["kaynak_ifade"])}
}

func mapStopSale(arr []any) []repository.StopSaleRange {
	out := []repository.StopSaleRange{}
	for _, item := range arr {
		m := asMap(item)
		if m == nil {
			continue
		}
		out = append(out, repository.StopSaleRange{
			Start:              strPtr(m["baslangic"]),
			End:                strPtr(m["bitis"]),
			Scope:              strPtr(m["kapsam"]),
			NotificationMethod: strPtr(m["bildirim_yontemi"]),
			SourcePhrase:       strPtr(m["kaynak_ifade"]),
		})
	}
	return out
}

func mapChildren(arr []any) []repository.ChildPolicy {
	var out []repository.ChildPolicy
	for _, item := range arr {
		m := asMap(item)
		if m == nil {
			continue
		}
		out = append(out, repository.ChildPolicy{
			AgeMin:          floatPtr(m["yas_min"]),
			AgeMax:          floatPtr(m["yas_max"]),
			DiscountPercent: floatPtr(m["indirim_yuzde"]),
			Free:            boolPtr(m["ucretsiz"]),
			Condition:       strPtr(m["kosul"]),
		})
	}
	return out
}

func mapCancel(arr []any) []repository.CancellationTerm {
	var out []repository.CancellationTerm
	for _, item := range arr {
		m := asMap(item)
		if m == nil {
			continue
		}
		out = append(out, repository.CancellationTerm{
			Scope:            strPtr(m["kapsam"]),
			Days:             int32Ptr(m["gun"]),
			CompensationNote: strPtr(m["tazminat_aciklama"]),
		})
	}
	return out
}

func mapNoShow(m map[string]any) *repository.NoShow {
	if m == nil {
		return nil
	}
	return &repository.NoShow{ResponsibleParty: strPtr(m["sorumlu_taraf"]), CompensationNote: strPtr(m["tazminat_aciklama"])}
}

func mapOverbooking(m map[string]any) *repository.Overbooking {
	if m == nil {
		return nil
	}
	return &repository.Overbooking{ResponsibleParty: strPtr(m["sorumlu_taraf"]), Description: strPtr(m["aciklama"])}
}

func mapPayment(m map[string]any) *repository.Payment {
	if m == nil {
		return nil
	}
	return &repository.Payment{
		DaysAfterInvoice: int32Ptr(m["fatura_sonrasi_gun"]),
		HasAdvance:       boolPtr(m["avans_var"]),
		AdvanceNote:      strPtr(m["avans_aciklama"]),
	}
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func asString(v any) (string, bool) {
	if v == nil {
		return "", false
	}
	switch x := v.(type) {
	case string:
		return x, true
	default:
		return fmt.Sprint(x), true
	}
}

func strPtr(v any) *string {
	s, ok := asString(v)
	if !ok || s == "" || s == "<nil>" {
		return nil
	}
	return &s
}

func asInt32(v any) (int32, bool) {
	switch n := v.(type) {
	case int:
		return int32(n), true
	case int32:
		return n, true
	case int64:
		return int32(n), true
	case float64:
		return int32(n), true
	case float32:
		return int32(n), true
	default:
		return 0, false
	}
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func int32Ptr(v any) *int32 {
	n, ok := asInt32(v)
	if !ok {
		return nil
	}
	return &n
}

func floatPtr(v any) *float64 {
	n, ok := asFloat(v)
	if !ok {
		return nil
	}
	return &n
}

func boolPtr(v any) *bool {
	b, ok := v.(bool)
	if !ok {
		return nil
	}
	return &b
}

func storedFileIDs(docs []repository.Contract) []string {
	var ids []string
	for i := range docs {
		id := strings.TrimSpace(docs[i].StoredFileID)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
