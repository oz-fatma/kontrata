"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  AktifPromptDocument,
  AyarlarDocument,
  AyarlariGuncelleDocument,
  LlmCagrilariDocument,
  LlmMetrikleriDocument,
  PromptGuncelleDocument,
  PromptSurumeDonDocument,
  PromptSurumleriDocument,
  UyelerDocument,
  type LlmCagrilariQuery,
  type LlmMetrikleriQuery,
  type PromptSurumleriQuery,
} from "@/generated/graphql";
import { PromptTipi } from "@/lib/enums";
import { useAuth } from "@/lib/auth";
import { AuthExpiredError, gqlRequest, graphqlMessage } from "@/lib/client";
import { formatDateTime } from "@/lib/format";
import { AppShell } from "@/components/shell";
import { EmptyState, ErrorState, Field, LoadingState } from "@/components/states";

type PromptKind = (typeof PromptTipi)[keyof typeof PromptTipi];
type PromptRow = PromptSurumleriQuery["promptSurumleri"][number];
type PanelTab = PromptKind | "METRIKLER";
type Metrics = LlmMetrikleriQuery["llmMetrikleri"];
type CallRow = LlmCagrilariQuery["llmCagrilari"][number];

const tabs: { id: PanelTab; label: string }[] = [
  { id: PromptTipi.Okuyucu, label: "Okuyucu" },
  { id: PromptTipi.Denetci, label: "Denetçi" },
  { id: "METRIKLER", label: "Metrikler" },
];

export default function AdminSettingsPage() {
  return (
    <AppShell>
      <AdminBody />
    </AppShell>
  );
}

function AdminBody() {
  const { org, isOwner, userId } = useAuth();
  const [tab, setTab] = useState<PanelTab>(PromptTipi.Okuyucu);
  const [draft, setDraft] = useState("");
  const [history, setHistory] = useState<PromptRow[] | null>(null);
  const [risk, setRisk] = useState("0.75");
  const [maxToken, setMaxToken] = useState("600");
  const [emails, setEmails] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [calls, setCalls] = useState<CallRow[] | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      if (tab === "METRIKLER") {
        const [m, c] = await Promise.all([
          gqlRequest(LlmMetrikleriDocument, { sonSaat: 24 }),
          gqlRequest(LlmCagrilariDocument, { limit: 20 }),
        ]);
        setMetrics(m.llmMetrikleri);
        setCalls(c.llmCagrilari);
        return;
      }
      const [aktif, liste, ayar, uyeler] = await Promise.all([
        gqlRequest(AktifPromptDocument, { tip: tab }),
        gqlRequest(PromptSurumleriDocument, { tip: tab }),
        gqlRequest(AyarlarDocument),
        gqlRequest(UyelerDocument),
      ]);
      setDraft(aktif.aktifPrompt.icerik);
      setHistory(liste.promptSurumleri);
      setRisk(String(ayar.ayarlar.denetciRiskEsigi));
      setMaxToken(String(ayar.ayarlar.maxToken));
      const map: Record<string, string> = {};
      for (const u of uyeler.uyeler) {
        map[u.id] = u.eposta;
      }
      setEmails(map);
    } catch (err) {
      if (err instanceof AuthExpiredError) {
        return;
      }
      setError(graphqlMessage(err));
      if (tab === "METRIKLER") {
        setMetrics(null);
        setCalls([]);
      } else {
        setHistory([]);
      }
    }
  }, [tab]);

  useEffect(() => {
    if (isOwner && org) {
      void load();
    }
  }, [isOwner, org, load]);

  const creatorLabel = useMemo(() => {
    return (id: string) => emails[id] || (id ? id.slice(-6) : "kod varsayılanı");
  }, [emails]);

  if (!org || !isOwner) {
    return (
      <EmptyState
        title="Yönetici paneli yok"
        detail="Prompt ve model ayarları yalnızca kurumsal hesap sahibine açıktır."
      />
    );
  }

  return (
    <div className="flex flex-col gap-[var(--space-section)]">
      <div>
        <h1>Yönetici</h1>
        <p className="meta-text mt-1">{org.ad}</p>
      </div>

      <p className="card bg-[var(--yellow-bg)] px-[var(--space-card)] py-3 text-[13px] text-[var(--yellow-ink)]">
        Prompt değişiklikleri tüm yeni çıkarımları etkiler. Kişisel veri maskelemesi
        prompt metninden bağımsız olarak her zaman çalışır.
      </p>

      {error ? <ErrorState message={error} onRetry={() => void load()} /> : null}

      <div className="flex gap-2">
        {tabs.map((item) => (
          <button
            key={item.id}
            type="button"
            className={tab === item.id ? "btn btn-primary" : "btn"}
            onClick={() => setTab(item.id)}
          >
            {item.label}
          </button>
        ))}
      </div>

      {tab === "METRIKLER" ? (
        <MetricsPanel metrics={metrics} calls={calls} />
      ) : (
        <>
      <section>
        <h2 className="mb-2">Aktif prompt</h2>
        {history === null ? <LoadingState /> : (
          <>
            <label htmlFor="prompt-icerik" className="sr-only">
              Prompt metni
            </label>
            <textarea
              id="prompt-icerik"
              rows={14}
              className="font-mono text-[13px]"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
            />
            <button
              type="button"
              className="btn btn-primary mt-2"
              disabled={busy || !draft.trim()}
              onClick={async () => {
                setBusy(true);
                setError(null);
                try {
                  await gqlRequest(PromptGuncelleDocument, { tip: tab as PromptKind, icerik: draft });
                  await load();
                } catch (err) {
                  setError(graphqlMessage(err));
                } finally {
                  setBusy(false);
                }
              }}
            >
              Yeni sürüm oluştur
            </button>
          </>
        )}
      </section>

      <section>
        <h2 className="mb-2">Sürüm geçmişi</h2>
        {history === null ? <LoadingState /> : null}
        {history && history.length === 0 ? (
          <EmptyState title="Kayıtlı sürüm yok" detail="Kod varsayılanı kullanılıyor." />
        ) : null}
        {history && history.length > 0 ? (
          <ul className="card list-panel divide-y divide-[var(--border)] overflow-hidden">
            {history.map((row) => (
              <li key={row.id} className="flex flex-wrap items-center justify-between gap-2 text-[14px]">
                <div>
                  <p className="font-medium text-[var(--ink)]">
                    Sürüm {row.surum}
                    {row.aktif ? (
                      <span className="ml-2 meta-text font-normal text-[var(--green)]">aktif</span>
                    ) : null}
                  </p>
                  <p className="meta-text tabular-nums">
                    {formatDateTime(row.olusturmaTarihi)}
                    {" · "}
                    {row.olusturanKullaniciId === userId
                      ? "siz"
                      : creatorLabel(row.olusturanKullaniciId)}
                  </p>
                </div>
                {!row.aktif ? (
                  <button
                    type="button"
                    className="btn"
                    disabled={busy}
                    onClick={async () => {
                      setBusy(true);
                      setError(null);
                      try {
                        await gqlRequest(PromptSurumeDonDocument, { id: row.id });
                        await load();
                      } catch (err) {
                        setError(graphqlMessage(err));
                      } finally {
                        setBusy(false);
                      }
                    }}
                  >
                    Bu sürüme dön
                  </button>
                ) : null}
              </li>
            ))}
          </ul>
        ) : null}
      </section>

      <section className="max-w-sm">
        <h2 className="mb-2">Model ayarları</h2>
        <form
          className="flex flex-col gap-3"
          onSubmit={async (ev) => {
            ev.preventDefault();
            const esik = Number(risk);
            const token = Number(maxToken);
            if (!Number.isFinite(esik) || esik < 0 || esik > 1) {
              setError("Risk eşiği 0 ile 1 arasında olmalı");
              return;
            }
            if (!Number.isFinite(token) || token < 1 || token > 8192) {
              setError("max token 1–8192 arasında olmalı");
              return;
            }
            setBusy(true);
            setError(null);
            try {
              await gqlRequest(AyarlariGuncelleDocument, {
                denetciRiskEsigi: esik,
                maxToken: Math.round(token),
              });
              await load();
            } catch (err) {
              setError(graphqlMessage(err));
            } finally {
              setBusy(false);
            }
          }}
        >
          <Field id="risk" label="Denetçi risk eşiği">
            <input
              id="risk"
              type="number"
              step="0.01"
              min={0}
              max={1}
              value={risk}
              onChange={(e) => setRisk(e.target.value)}
            />
          </Field>
          <Field id="max-token" label="max token">
            <input
              id="max-token"
              type="number"
              min={1}
              max={8192}
              value={maxToken}
              onChange={(e) => setMaxToken(e.target.value)}
            />
          </Field>
          <button type="submit" className="btn btn-primary" disabled={busy}>
            Ayarları kaydet
          </button>
        </form>
      </section>
        </>
      )}
    </div>
  );
}

function pct(n: number): string {
  return `%${Math.round(n * 100)}`;
}

function ms(n: number): string {
  return `${Math.round(n)} ms`;
}

function hataLabel(tip: string): string {
  switch (tip) {
    case "zaman_asimi":
      return "zaman aşımı";
    case "http_5xx":
      return "HTTP 5xx";
    case "http_4xx":
      return "HTTP 4xx";
    case "ayristirma":
      return "ayrıştırma";
    case "yok":
      return "yok";
    default:
      return tip;
  }
}

function MetricsPanel({
  metrics,
  calls,
}: {
  metrics: Metrics | null;
  calls: CallRow[] | null;
}) {
  if (metrics === null && calls === null) {
    return <LoadingState />;
  }
  const m = metrics;
  return (
    <div className="flex flex-col gap-8">
      <section>
        <h2 className="mb-2">Son 24 saat</h2>
        {m ? (
          <dl className="card grid max-w-xl grid-cols-2 gap-4 p-[var(--space-card)] text-[14px] sm:grid-cols-3">
            <div>
              <dt className="meta-text">Toplam çağrı</dt>
              <dd className="mt-0.5 font-medium tabular-nums text-[var(--ink)]">{m.toplamCagri}</dd>
            </div>
            <div>
              <dt className="meta-text">Başarılı</dt>
              <dd className="mt-0.5 font-medium tabular-nums text-[var(--ink)]">{m.basariliCagri}</dd>
            </div>
            <div>
              <dt className="meta-text">Başarısız</dt>
              <dd className="mt-0.5 font-medium tabular-nums text-[var(--ink)]">{m.basarisizCagri}</dd>
            </div>
            <div>
              <dt className="meta-text">Ortalama süre</dt>
              <dd className="mt-0.5 font-medium tabular-nums text-[var(--ink)]">{ms(m.ortalamaSureMs)}</dd>
            </div>
            <div>
              <dt className="meta-text">p95 süre</dt>
              <dd className="mt-0.5 font-medium tabular-nums text-[var(--ink)]">{ms(m.p95SureMs)}</dd>
            </div>
          </dl>
        ) : (
          <EmptyState title="Metrik yok" detail="Henüz LLM çağrısı kaydı yok." />
        )}
      </section>

      <section>
        <h2 className="mb-2">Agent</h2>
        {m && m.agentBazinda.length > 0 ? (
          <div className="card max-w-xl overflow-x-auto">
            <table className="data-table w-full text-left text-[14px]">
              <thead>
                <tr>
                  <th className="font-medium">Agent</th>
                  <th className="font-medium">Çağrı</th>
                  <th className="font-medium">Ort. süre</th>
                  <th className="font-medium">Başarı</th>
                </tr>
              </thead>
              <tbody>
                {m.agentBazinda.map((row) => (
                  <tr key={row.agent}>
                    <td>{row.agent === "DENETCI" ? "Denetçi" : "Okuyucu"}</td>
                    <td className="tabular-nums">{row.cagri}</td>
                    <td className="tabular-nums">{ms(row.ortalamaSureMs)}</td>
                    <td className="tabular-nums">{pct(row.basariOrani)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="text-[14px] text-[var(--ink-muted)]">Kayıt yok.</p>
        )}
      </section>

      <section>
        <h2 className="mb-2">Uç</h2>
        {m && m.ucBazinda.length > 0 ? (
          <div className="card max-w-xl overflow-x-auto">
            <table className="data-table w-full text-left text-[14px]">
              <thead>
                <tr>
                  <th className="font-medium">Uç</th>
                  <th className="font-medium">Çağrı</th>
                  <th className="font-medium">Ort. süre</th>
                  <th className="font-medium">Başarı</th>
                </tr>
              </thead>
              <tbody>
                {m.ucBazinda.map((row) => (
                  <tr key={row.ucAdi}>
                    <td>{row.ucAdi}</td>
                    <td className="tabular-nums">{row.cagri}</td>
                    <td className="tabular-nums">{ms(row.ortalamaSureMs)}</td>
                    <td className="tabular-nums">{pct(row.basariOrani)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="text-[14px] text-[var(--ink-muted)]">Kayıt yok.</p>
        )}
      </section>

      <section>
        <h2 className="mb-2">Hata dağılımı</h2>
        {m && m.hataDagilimi.length > 0 ? (
          <ul className="text-[13px]">
            {m.hataDagilimi.map((row) => (
              <li key={row.hataTipi}>
                {hataLabel(row.hataTipi)}: {row.adet}
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-[13px] text-[var(--muted)]">Hata yok.</p>
        )}
      </section>

      <section>
        <h2 className="mb-2">Son çağrılar</h2>
        {calls && calls.length > 0 ? (
          <ul className="card list-panel divide-y divide-[var(--border)] overflow-hidden">
            {calls.map((row, i) => (
              <li
                key={`${row.baslangic}-${i}`}
                className="flex flex-wrap items-center justify-between gap-2 text-[14px]"
              >
                <span className="tabular-nums text-[var(--ink)]">
                  {row.agent === "DENETCI" ? "Denetçi" : "Okuyucu"} · {row.ucAdi} · {row.sureMs} ms
                </span>
                <span className={row.basarili ? "text-[var(--green)]" : "text-[var(--red)]"}>
                  {row.basarili ? "başarılı" : hataLabel(row.hataTipi)}
                </span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-[14px] text-[var(--ink-muted)]">Çağrı yok.</p>
        )}
      </section>
    </div>
  );
}
