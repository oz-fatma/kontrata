"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  AktifPromptDocument,
  AyarlarDocument,
  AyarlariGuncelleDocument,
  PromptGuncelleDocument,
  PromptSurumeDonDocument,
  PromptSurumleriDocument,
  UyelerDocument,
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

const tabs: { id: PromptKind; label: string }[] = [
  { id: PromptTipi.Okuyucu, label: "Okuyucu" },
  { id: PromptTipi.Denetci, label: "Denetçi" },
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
  const [tab, setTab] = useState<PromptKind>(PromptTipi.Okuyucu);
  const [draft, setDraft] = useState("");
  const [history, setHistory] = useState<PromptRow[] | null>(null);
  const [risk, setRisk] = useState("0.75");
  const [maxToken, setMaxToken] = useState("600");
  const [emails, setEmails] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setError(null);
    try {
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
      setHistory([]);
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
    <div className="flex flex-col gap-8">
      <div>
        <h1>Yönetici</h1>
        <p className="text-[12px] text-[var(--muted)]">{org.ad}</p>
      </div>

      <p className="rounded-card border-[0.5px] border-[var(--line)] bg-[var(--yellow-bg)] px-3 py-2 text-[13px] text-[var(--yellow-ink)]">
        Prompt değişiklikleri tüm yeni çıkarımları etkiler. Kişisel veri maskelemesi
        prompt'tan bağımsız olarak her zaman çalışır.
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
                  await gqlRequest(PromptGuncelleDocument, { tip: tab, icerik: draft });
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
          <ul className="divide-y divide-[var(--line)] rounded-card border-[0.5px] border-[var(--line)]">
            {history.map((row) => (
              <li key={row.id} className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 text-[13px]">
                <div>
                  <p className="font-medium">
                    Sürüm {row.surum}
                    {row.aktif ? (
                      <span className="ml-2 text-[12px] font-normal text-[var(--green)]">aktif</span>
                    ) : null}
                  </p>
                  <p className="text-[12px] text-[var(--muted)]">
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
    </div>
  );
}
