"use client";

import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import {
  SozlesmeDocument,
  SozlesmeGuncelleDocument,
  type SozlesmeQuery,
} from "@/generated/graphql";
import { SozlesmeDurumu } from "@/lib/enums";
import { useAuth } from "@/lib/auth";
import { AuthExpiredError, gqlRequest, graphqlMessage } from "@/lib/client";
import {
  enumLabel,
  formatDateTime,
  formatPeriod,
  missingField,
  statusLabel,
  statusTone,
} from "@/lib/format";
import { AppShell } from "@/components/shell";
import { EmptyState, ErrorState, LoadingState } from "@/components/states";
import { StatusBadge } from "@/components/status-badge";
import { ExtractedField } from "@/components/extracted-field";
import { FindingCard } from "@/components/finding-card";

type Contract = NonNullable<SozlesmeQuery["sozlesme"]>;

export default function ContractDetailPage() {
  return (
    <AppShell>
      <ContractDetail />
    </AppShell>
  );
}

function ContractDetail() {
  const params = useSearchParams();
  const id = params.get("id") ?? "";
  const { canWrite } = useAuth();
  const [row, setRow] = useState<Contract | null | undefined>(undefined);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function load() {
    if (!id) {
      setRow(null);
      return;
    }
    setError(null);
    try {
      const data = await gqlRequest(SozlesmeDocument, { id });
      setRow(data.sozlesme ?? null);
    } catch (err) {
      if (err instanceof AuthExpiredError) {
        return;
      }
      setError(graphqlMessage(err));
      setRow(null);
    }
  }

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  useEffect(() => {
    const waiting =
      row?.durum === SozlesmeDurumu.Isleniyor ||
      row?.durum === SozlesmeDurumu.Yuklendi;
    if (!waiting) {
      return;
    }
    const t = window.setInterval(() => {
      void load();
    }, 3000);
    return () => window.clearInterval(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, row?.durum]);

  const fields = useMemo(() => (row ? fieldRows(row) : []), [row]);

  async function approve() {
    if (!row) {
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await gqlRequest(SozlesmeGuncelleDocument, {
        id: row.id,
        girdi: { durum: SozlesmeDurumu.Onaylandi },
      });
      await load();
    } catch (err) {
      setError(graphqlMessage(err));
    } finally {
      setBusy(false);
    }
  }

  function downloadJson() {
    if (!row) {
      return;
    }
    const blob = new Blob([JSON.stringify(row, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${row.dosyaAdi || "sozlesme"}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }

  if (row === undefined && !error) {
    return <LoadingState />;
  }
  if (error) {
    return <ErrorState message={error} onRetry={() => void load()} />;
  }
  if (!id) {
    return (
      <EmptyState
        title="Sözleşme seçilmedi"
        detail="Listeden bir kayıt açın."
      />
    );
  }
  if (!row) {
    return (
      <EmptyState
        title="Sözleşme bulunamadı"
        detail="Kayıt silinmiş olabilir veya henüz yüklenmedi."
      />
    );
  }

  const processing =
    row.durum === SozlesmeDurumu.Isleniyor ||
    row.durum === SozlesmeDurumu.Yuklendi;

  return (
    <div>
      <div className="mb-4 flex items-start justify-between gap-3">
        <div>
          <h1>{row.dosyaAdi || "Adsız dosya"}</h1>
          <p className="text-[12px] text-[var(--muted)]">
            — sayfa · {formatDateTime(row.guncellemeTarihi)}
          </p>
        </div>
        <StatusBadge label={statusLabel(row.durum)} tone={statusTone(row.durum)} />
      </div>

      <div className="grid gap-4 md:grid-cols-[1fr_16rem]">
        <section className="rounded-card border-[0.5px] border-[var(--line)]">
          {processing ? (
            <p className="px-3 py-6 text-[13px] text-[var(--muted)]">
              Sözleşme işleniyor. Çıkarım birkaç dakika sürebilir.
            </p>
          ) : (
            fields.map((f) => (
              <ExtractedField
                key={f.path}
                label={f.label}
                path={f.path}
                value={f.value}
                metas={row.cikarimMeta}
              />
            ))
          )}
        </section>

        <aside className="bg-[var(--subtle)] px-3 py-3">
          <h2 className="mb-2">Denetçi bulguları</h2>
          <FindingCard
            tone="red"
            title="Kritik madde"
            body="Denetçi çıktısı Aşama 8’de bağlanacak."
          />
          <FindingCard
            tone="yellow"
            title="Uyarı"
            body="Denetçi çıktısı Aşama 8’de bağlanacak."
          />
          <div className="mt-3 border-t-[0.5px] border-[var(--line)] pt-2 text-[12px] text-[var(--muted)]">
            <p>0 bulgu</p>
            <p>
              Okuyucu süresi{" "}
              {row.islemSuresi != null
                ? `${Math.round(row.islemSuresi)} sn`
                : "—"}
            </p>
            <p>Denetçi süresi —</p>
            {row.semaHatalari && row.semaHatalari.length > 0 ? (
              <p>Şema hataları: {row.semaHatalari.length}</p>
            ) : null}
          </div>
        </aside>
      </div>

      <div className="mt-4 flex flex-wrap gap-2">
        {canWrite && row.durum !== SozlesmeDurumu.Onaylandi ? (
          <button
            type="button"
            className="btn btn-primary"
            disabled={busy}
            onClick={() => void approve()}
          >
            Onayla
          </button>
        ) : null}
        <button type="button" className="btn" disabled title="Kaynak PDF henüz bağlı değil">
          Kaynağı aç
        </button>
        <button type="button" className="btn" onClick={downloadJson}>
          JSON indir
        </button>
      </div>
    </div>
  );
}

function fieldRows(row: Contract): { label: string; path: string; value: string | null }[] {
  const kontenjan = row.odaKontenjanlari?.length
    ? row.odaKontenjanlari
        .map((k) => `${k.odaTipi}: ${k.adet}${k.aciklama ? ` (${k.aciklama})` : ""}`)
        .join("\n")
    : null;
  const fiyat = row.fiyatlar?.length
    ? row.fiyatlar
        .map(
          (f) =>
            `${f.odaTipi} · ${f.pansiyon ? enumLabel(f.pansiyon) : "—"} · ${f.tutar} (${enumLabel(f.birim)})`,
        )
        .join("\n")
    : null;
  const stop = row.stopSale?.length
    ? row.stopSale
        .map((s) => `${formatPeriod(s.baslangic, s.bitis)} · ${s.kapsam ?? missingField()}`)
        .join("\n")
    : null;

  return [
    { label: "Otel", path: "meta.otelAdi", value: row.meta?.otelAdi ?? null },
    { label: "Operatör", path: "meta.acenteAdi", value: row.meta?.acenteAdi ?? null },
    {
      label: "Sözleşme tipi",
      path: "meta.sozlesmeTipi",
      value: row.meta?.sozlesmeTipi ? enumLabel(row.meta.sozlesmeTipi) : null,
    },
    {
      label: "Sezon",
      path: "meta.sezon",
      value: row.meta?.sezon ? enumLabel(row.meta.sezon) : null,
    },
    { label: "Para birimi", path: "meta.paraBirimi", value: row.meta?.paraBirimi ?? null },
    {
      label: "Kur esası",
      path: "meta.kurEsasi",
      value: row.meta?.kurEsasi ? enumLabel(row.meta.kurEsasi) : null,
    },
    {
      label: "Yetkili mahkeme",
      path: "meta.yetkiliMahkeme",
      value: row.meta?.yetkiliMahkeme ?? null,
    },
    { label: "İmza tarihi", path: "meta.imzaTarihi", value: row.meta?.imzaTarihi ?? null },
    {
      label: "Dönem",
      path: "donem",
      value:
        row.donem?.baslangic || row.donem?.bitis
          ? formatPeriod(row.donem?.baslangic, row.donem?.bitis)
          : null,
    },
    { label: "Oda kontenjanı", path: "odaKontenjanlari", value: kontenjan },
    { label: "Fiyatlar", path: "fiyatlar", value: fiyat },
    {
      label: "Release",
      path: "release",
      value: row.release
        ? `${row.release.gun} gün · ${enumLabel(row.release.kapsam)}`
        : null,
    },
    { label: "Stop sale", path: "stopSale", value: stop },
  ];
}
