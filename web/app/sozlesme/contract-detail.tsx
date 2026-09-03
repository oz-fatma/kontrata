"use client";

import { FileText, Inbox, SearchX } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import {
  SozlesmeAlanGuncelleDocument,
  SozlesmeDocument,
  SozlesmeOnaylaDocument,
  type SozlesmeQuery,
} from "@/generated/graphql";
import { SozlesmeDurumu } from "@/lib/enums";
import { useAuth } from "@/lib/auth";
import {
  AuthExpiredError,
  fetchContractFile,
  gqlRequest,
  graphqlMessage,
} from "@/lib/client";
import {
  enumLabel,
  extractionJsonName,
  findingSourceLabel,
  findingTone,
  formatDateTime,
  formatPeriod,
  isExtractPending,
  missingField,
  statusLabel,
  statusTone,
} from "@/lib/format";
import { usePolling } from "@/lib/use-polling";
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

  const load = useCallback(
    async (opts?: { silent?: boolean }) => {
      if (!id) {
        setRow(null);
        return;
      }
      if (!opts?.silent) {
        setError(null);
      }
      try {
        const data = await gqlRequest(SozlesmeDocument, { id });
        setRow(data.sozlesme ?? null);
      } catch (err) {
        if (err instanceof AuthExpiredError) {
          return;
        }
        if (!opts?.silent) {
          setError(graphqlMessage(err));
          setRow(null);
        }
      }
    },
    [id],
  );

  useEffect(() => {
    void load();
  }, [load]);

  usePolling(() => load({ silent: true }), isExtractPending(row?.durum));

  const fields = useMemo(() => (row ? fieldRows(row) : []), [row]);
  const approved = row?.durum === SozlesmeDurumu.Onaylandi;
  const canApprove = Boolean(canWrite && row?.durum === SozlesmeDurumu.IncelenmeyiBekliyor);
  const canEdit = Boolean(canWrite && row && !approved && !isExtractPending(row.durum));

  async function approve() {
    if (!row) {
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const data = await gqlRequest(SozlesmeOnaylaDocument, { id: row.id });
      setRow((prev) => (prev ? { ...prev, ...data.sozlesmeOnayla } : prev));
      await load();
    } catch (err) {
      setError(graphqlMessage(err));
    } finally {
      setBusy(false);
    }
  }

  async function saveField(path: string, value: string) {
    if (!row) {
      return;
    }
    setError(null);
    try {
      const data = await gqlRequest(SozlesmeAlanGuncelleDocument, {
        id: row.id,
        alanYolu: path,
        deger: value,
      });
      setRow((prev) => (prev ? { ...prev, ...data.sozlesmeAlanGuncelle } : prev));
    } catch (err) {
      setError(graphqlMessage(err));
    }
  }

  function downloadJson() {
    if (!row) {
      return;
    }
    const payload = {
      meta: row.meta,
      donem: row.donem,
      odaKontenjanlari: row.odaKontenjanlari,
      fiyatlar: row.fiyatlar,
      release: row.release,
      stopSale: row.stopSale,
      cikarimMeta: row.cikarimMeta,
      bulgular: row.bulgular,
      semaHatalari: row.semaHatalari,
    };
    const blob = new Blob([JSON.stringify(payload, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = extractionJsonName(row.dosyaAdi);
    a.click();
    URL.revokeObjectURL(url);
  }

  async function openSource() {
    if (!row) {
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const blob = await fetchContractFile(row.id);
      const url = URL.createObjectURL(blob);
      const opened = window.open(url, "_blank", "noopener,noreferrer");
      if (!opened) {
        const a = document.createElement("a");
        a.href = url;
        a.target = "_blank";
        a.rel = "noopener noreferrer";
        a.click();
      }
      window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
    } catch (err) {
      if (err instanceof AuthExpiredError) {
        return;
      }
      setError(err instanceof Error ? err.message : graphqlMessage(err));
    } finally {
      setBusy(false);
    }
  }

  if (row === undefined && !error) {
    return <LoadingState />;
  }
  if (error && !row) {
    return <ErrorState message={error} onRetry={() => void load()} />;
  }
  if (!id) {
    return (
      <EmptyState
        icon={SearchX}
        title="Sözleşme seçilmedi"
        detail="Listeden bir kayıt açın."
      />
    );
  }
  if (!row) {
    return (
      <EmptyState
        icon={FileText}
        title="Sözleşme bulunamadı"
        detail="Kayıt silinmiş olabilir veya henüz yüklenmedi."
      />
    );
  }

  const processing = isExtractPending(row.durum);
  const findings = row.bulgular ?? [];

  return (
    <div>
      <div className="mb-[var(--space-card-gap)] flex items-start justify-between gap-4">
        <div>
          <h1>{row.dosyaAdi || "Adsız dosya"}</h1>
          <p className="meta-text mt-1 tabular-nums">
            — sayfa · {formatDateTime(row.guncellemeTarihi)}
          </p>
        </div>
        <StatusBadge
          label={statusLabel(row.durum)}
          tone={statusTone(row.durum)}
          durum={row.durum}
        />
      </div>

      {error ? <ErrorState message={error} onRetry={() => void load()} /> : null}

      <div className="grid gap-[var(--space-card-gap)] md:grid-cols-[1fr_16rem]">
        <section className="card overflow-hidden">
          {processing ? (
            <p className="px-[var(--space-card)] py-6 text-[14px] text-[var(--ink-muted)]">
              Sözleşme işleniyor. Çıkarım birkaç dakika sürebilir.
            </p>
          ) : (
            fields.map((f) => (
              <ExtractedField
                key={f.path}
                label={f.label}
                path={f.path}
                lines={f.lines}
                metas={row.cikarimMeta}
                readOnly={!canEdit}
                listStyle={f.listStyle}
                onSave={canEdit ? saveField : undefined}
              />
            ))
          )}
        </section>

        <aside className="card px-[var(--space-card)] py-[var(--space-card)]">
          <h2 className="mb-3">Denetçi bulguları</h2>
          {processing ? (
            <p className="text-[14px] text-[var(--ink-muted)]">Denetçi henüz çalışmadı.</p>
          ) : findings.length === 0 ? (
            <EmptyState compact icon={Inbox} title="Bulgu yok" detail="Denetçi uyarı üretmedi." />
          ) : (
            findings.map((f) => (
              <FindingCard
                key={`${f.kod}-${f.alanYolu ?? ""}-${f.baslik}`}
                tone={findingTone(f.onem)}
                title={f.baslik}
                body={f.aciklama}
                source={findingSourceLabel(f.kaynak)}
              />
            ))
          )}
          <div className="mt-4 border-t-[0.5px] border-[var(--border)] pt-3 meta-text tabular-nums">
            <p>{processing ? "—" : `${findings.length} bulgu`}</p>
            <p>
              Okuyucu süresi{" "}
              {row.islemSuresi != null
                ? `${Math.round(row.islemSuresi)} sn`
                : "—"}
            </p>
            <p>
              Denetçi süresi{" "}
              {row.denetciSuresi != null ? `${row.denetciSuresi} sn` : "—"}
            </p>
            {row.semaHatalari && row.semaHatalari.length > 0 ? (
              <p>Şema hataları: {row.semaHatalari.length}</p>
            ) : null}
          </div>
        </aside>
      </div>

      <div className="mt-[var(--space-card-gap)] flex flex-wrap gap-3">
        {canApprove ? (
          <button
            type="button"
            className="btn btn-primary"
            disabled={busy}
            onClick={() => void approve()}
          >
            Onayla
          </button>
        ) : approved ? (
          <button type="button" className="btn btn-primary" disabled>
            Onaylandı
          </button>
        ) : null}
        <button
          type="button"
          className="btn"
          disabled={busy}
          onClick={() => void openSource()}
        >
          Kaynağı aç
        </button>
        <button type="button" className="btn" onClick={downloadJson}>
          JSON indir
        </button>
      </div>
    </div>
  );
}

function fieldRows(
  row: Contract,
): { label: string; path: string; lines: string[]; listStyle?: boolean }[] {
  const kontenjan = (row.odaKontenjanlari ?? []).map(
    (k) => `${k.odaTipi}: ${k.adet}${k.aciklama ? ` (${k.aciklama})` : ""}`,
  );
  const fiyat = (row.fiyatlar ?? []).map(
    (f) =>
      `${f.odaTipi} · ${f.pansiyon ? enumLabel(f.pansiyon) : "—"} · ${f.tutar} (${enumLabel(f.birim)})`,
  );
  const stop = (row.stopSale ?? []).map(
    (s) => `${formatPeriod(s.baslangic, s.bitis)} · ${s.kapsam ?? missingField()}`,
  );
  const donem =
    row.donem?.baslangic || row.donem?.bitis
      ? [formatPeriod(row.donem?.baslangic, row.donem?.bitis)]
      : [];
  const release = row.release
    ? [`${row.release.gun} gün · ${enumLabel(row.release.kapsam)}`]
    : [];

  return [
    { label: "Otel", path: "meta.otelAdi", lines: compact(row.meta?.otelAdi) },
    { label: "Operatör", path: "meta.acenteAdi", lines: compact(row.meta?.acenteAdi) },
    {
      label: "Sözleşme tipi",
      path: "meta.sozlesmeTipi",
      lines: compact(row.meta?.sozlesmeTipi ? enumLabel(row.meta.sozlesmeTipi) : null),
    },
    {
      label: "Sezon",
      path: "meta.sezon",
      lines: compact(row.meta?.sezon ? enumLabel(row.meta.sezon) : null),
    },
    { label: "Para birimi", path: "meta.paraBirimi", lines: compact(row.meta?.paraBirimi) },
    {
      label: "Kur esası",
      path: "meta.kurEsasi",
      lines: compact(row.meta?.kurEsasi ? enumLabel(row.meta.kurEsasi) : null),
    },
    {
      label: "Yetkili mahkeme",
      path: "meta.yetkiliMahkeme",
      lines: compact(row.meta?.yetkiliMahkeme),
    },
    { label: "İmza tarihi", path: "meta.imzaTarihi", lines: compact(row.meta?.imzaTarihi) },
    { label: "Dönem", path: "donem", lines: donem },
    { label: "Oda kontenjanı", path: "odaKontenjanlari", lines: kontenjan, listStyle: true },
    { label: "Fiyatlar", path: "fiyatlar", lines: fiyat, listStyle: true },
    { label: "Release", path: "release", lines: release },
    { label: "Stop sale", path: "stopSale", lines: stop, listStyle: true },
  ];
}

function compact(value: string | null | undefined): string[] {
  const s = value?.trim();
  return s ? [s] : [];
}
