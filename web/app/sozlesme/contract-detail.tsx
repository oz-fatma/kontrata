"use client";

import { ArrowLeft, FileText, Inbox, Loader2, SearchX } from "lucide-react";
import Link from "next/link";
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

type FieldRow = {
  label: string;
  path: string;
  lines: string[];
  listStyle?: boolean;
};

type FieldSection = {
  title: string;
  fields: FieldRow[];
};

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

  const sections = useMemo(() => (row ? fieldSections(row) : []), [row]);
  const approved = row?.durum === SozlesmeDurumu.Onaylandi;
  const canApprove = Boolean(canWrite && row?.durum === SozlesmeDurumu.IncelenmeyiBekliyor);
  const canEdit = Boolean(
    canWrite && row && !approved && !isExtractPending(row.durum),
  );

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
  const subtitleParts = [
    row.meta?.acenteAdi || row.meta?.otelAdi || null,
    formatDateTime(row.guncellemeTarihi),
  ].filter(Boolean);

  return (
    <div>
      <div className="mb-[var(--space-card-gap)] flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <Link
            href="/"
            className="mb-2 inline-flex items-center gap-1.5 text-[13px] font-medium text-[var(--ink-muted)] hover:text-[var(--ink)]"
          >
            <ArrowLeft className="size-3.5 shrink-0" aria-hidden />
            Sözleşmelere dön
          </Link>
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="min-w-0 break-words">{row.dosyaAdi || "Adsız dosya"}</h1>
            <StatusBadge
              label={statusLabel(row.durum)}
              tone={statusTone(row.durum)}
              durum={row.durum}
            />
          </div>
          <p className="meta-text mt-1 tabular-nums">{subtitleParts.join(" · ")}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {canApprove ? (
            <button
              type="button"
              className="btn btn-primary"
              disabled={busy}
              onClick={() => void approve()}
            >
              Onayla
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

      {error ? <ErrorState message={error} onRetry={() => void load()} /> : null}

      <div className="grid gap-[var(--space-card-gap)] md:grid-cols-[1fr_minmax(16rem,20rem)]">
        <div className="flex flex-col gap-[var(--space-card-gap)]">
          {processing ? (
            <section className="card overflow-hidden">
              <div className="flex items-start gap-3 px-[var(--space-card)] py-8">
                <Loader2
                  className="mt-0.5 size-5 shrink-0 animate-spin text-[var(--blue)]"
                  aria-hidden
                />
                <div>
                  <p className="text-[14px] font-medium text-[var(--ink)]">
                    Sözleşme işleniyor
                  </p>
                  <p className="meta-text mt-1">
                    Okuyucu çıkarımı birkaç dakika sürebilir. Bu sayfa otomatik
                    yenilenir.
                  </p>
                </div>
              </div>
              <div className="border-t-[0.5px] border-[var(--border)] px-[var(--space-card)] py-4">
                <div className="flex animate-pulse flex-col gap-3" aria-hidden>
                  {[1, 2, 3, 4].map((i) => (
                    <div key={i} className="flex flex-col gap-2">
                      <div className="h-2.5 w-20 rounded bg-[var(--border)]/80" />
                      <div className="h-3.5 w-full max-w-sm rounded bg-[var(--border)]/50" />
                    </div>
                  ))}
                </div>
              </div>
            </section>
          ) : (
            sections.map((section) => (
              <section key={section.title} className="card overflow-hidden">
                <h2 className="border-b-[0.5px] border-[var(--border)] bg-[var(--surface-subtle)] px-[var(--space-card)] py-2.5 text-[13px] font-semibold text-[var(--ink-muted)]">
                  {section.title}
                </h2>
                {section.fields.map((f) => (
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
                ))}
              </section>
            ))
          )}
        </div>

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
    </div>
  );
}

function fieldSections(row: Contract): FieldSection[] {
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
    {
      title: "Taraflar ve meta",
      fields: [
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
      ],
    },
    {
      title: "Kontenjan ve fiyat",
      fields: [
        { label: "Oda kontenjanı", path: "odaKontenjanlari", lines: kontenjan, listStyle: true },
        { label: "Fiyatlar", path: "fiyatlar", lines: fiyat, listStyle: true },
      ],
    },
    {
      title: "Release ve stop sale",
      fields: [
        { label: "Release", path: "release", lines: release },
        { label: "Stop sale", path: "stopSale", lines: stop, listStyle: true },
      ],
    },
  ];
}

function compact(value: string | null | undefined): string[] {
  const s = value?.trim();
  return s ? [s] : [];
}
