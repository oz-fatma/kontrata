"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import {
  SozlesmeSilDocument,
  SozlesmeYukleDocument,
  SozlesmelerDocument,
  type SozlesmelerQuery,
} from "@/generated/graphql";
import { SozlesmeDurumu } from "@/lib/enums";
import { useAuth } from "@/lib/auth";
import { AuthExpiredError, gqlRequest, graphqlMessage } from "@/lib/client";
import {
  formatDateTime,
  formatPeriod,
  statusLabel,
  statusTone,
} from "@/lib/format";
import { AppShell } from "@/components/shell";
import { EmptyState, ErrorState, LoadingState } from "@/components/states";
import { StatusBadge } from "@/components/status-badge";

type Row = SozlesmelerQuery["sozlesmeler"][number];

export default function HomePage() {
  return (
    <AppShell>
      <ContractList />
    </AppShell>
  );
}

function ContractList() {
  const { org, canWrite } = useAuth();
  const router = useRouter();
  const fileRef = useRef<HTMLInputElement>(null);
  const [rows, setRows] = useState<Row[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("all");
  const [busy, setBusy] = useState(false);

  const load = async () => {
    setError(null);
    try {
      const data = await gqlRequest(SozlesmelerDocument, {
        limit: 100,
        offset: 0,
      });
      setRows(data.sozlesmeler);
    } catch (err) {
      if (err instanceof AuthExpiredError) {
        router.replace("/giris/");
        return;
      }
      setError(graphqlMessage(err));
    }
  };

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const filtered = useMemo(() => {
    if (!rows) {
      return [];
    }
    const q = query.trim().toLocaleLowerCase("tr");
    return rows.filter((r) => {
      if (status !== "all" && r.durum !== status) {
        return false;
      }
      if (!q) {
        return true;
      }
      const hay = `${r.dosyaAdi ?? ""} ${r.meta?.acenteAdi ?? ""} ${r.meta?.otelAdi ?? ""}`.toLocaleLowerCase("tr");
      return hay.includes(q);
    });
  }, [rows, query, status]);

  async function onUpload(file: File) {
    setBusy(true);
    setError(null);
    try {
      await gqlRequest(SozlesmeYukleDocument, { dosya: file });
      await load();
    } catch (err) {
      setError(graphqlMessage(err));
    } finally {
      setBusy(false);
    }
  }

  async function onDelete(id: string, e: React.MouseEvent) {
    e.stopPropagation();
    if (!window.confirm("Bu sözleşmeyi silmek istiyor musunuz?")) {
      return;
    }
    try {
      await gqlRequest(SozlesmeSilDocument, { id });
      await load();
    } catch (err) {
      setError(graphqlMessage(err));
    }
  }

  return (
    <div>
      <div className="mb-3 flex items-end justify-between gap-3">
        <div>
          <h1>Sözleşmeler</h1>
          <p className="text-[12px] text-[var(--muted)]">
            {org?.ad ?? "Bireysel hesap"}
            {rows ? ` · ${rows.length} kayıt` : ""}
          </p>
        </div>
        {canWrite ? (
          <div>
            <input
              ref={fileRef}
              id="sozlesme-yukle"
              type="file"
              accept="application/pdf"
              className="sr-only"
              onChange={(e) => {
                const f = e.target.files?.[0];
                e.target.value = "";
                if (f) {
                  void onUpload(f);
                }
              }}
            />
            <button
              type="button"
              className="btn btn-primary"
              disabled={busy}
              onClick={() => fileRef.current?.click()}
            >
              Sözleşme yükle
            </button>
          </div>
        ) : null}
      </div>

      <div className="mb-3 flex flex-wrap gap-2">
        <div className="min-w-[12rem] flex-1">
          <label htmlFor="arama" className="sr-only">
            Ara
          </label>
          <input
            id="arama"
            type="search"
            placeholder="Dosya veya operatör ara"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
        <div>
          <label htmlFor="durum-filtre" className="sr-only">
            Durum
          </label>
          <select
            id="durum-filtre"
            autoComplete="off"
            value={status}
            onChange={(e) => setStatus(e.target.value)}
          >
            <option value="all">Tüm durumlar</option>
            <option value={SozlesmeDurumu.Yuklendi}>Yüklendi</option>
            <option value={SozlesmeDurumu.Isleniyor}>İşleniyor</option>
            <option value={SozlesmeDurumu.IncelenmeyiBekliyor}>
              İncelenmeyi bekliyor
            </option>
            <option value={SozlesmeDurumu.Onaylandi}>Onaylandı</option>
            <option value={SozlesmeDurumu.Hata}>Hata</option>
          </select>
        </div>
      </div>

      {error ? <ErrorState message={error} onRetry={() => void load()} /> : null}
      {rows === null && !error ? <LoadingState /> : null}
      {rows && filtered.length === 0 ? (
        <EmptyState
          title="Sözleşme yok"
          detail="Yüklenen sözleşmeler burada listelenir."
        />
      ) : null}
      {rows && filtered.length > 0 ? (
        <div className="overflow-x-auto rounded-card border-[0.5px] border-[var(--line)]">
          <table className="w-full text-left text-[13px]">
            <thead className="border-b-[0.5px] border-[var(--line)] text-[12px] text-[var(--muted)]">
              <tr>
                <th className="px-3 py-2 font-medium">Dosya</th>
                <th className="px-3 py-2 font-medium">Dönem</th>
                <th className="px-3 py-2 font-medium">Bulgu</th>
                <th className="px-3 py-2 font-medium">Durum</th>
                {canWrite ? <th className="px-3 py-2 font-medium sr-only">İşlem</th> : null}
              </tr>
            </thead>
            <tbody>
              {filtered.map((row) => {
                const processing = row.durum === SozlesmeDurumu.Isleniyor;
                return (
                  <tr
                    key={row.id}
                    className="cursor-pointer border-b-[0.5px] border-[var(--line)] last:border-0 hover:bg-[var(--subtle)]"
                    onClick={() => router.push(`/sozlesme/?id=${encodeURIComponent(row.id)}`)}
                  >
                    <td className="px-3 py-2">
                      <div className="font-medium">{row.dosyaAdi || "Adsız dosya"}</div>
                      <div className="text-[12px] text-[var(--muted)]">
                        {row.meta?.acenteAdi || row.meta?.otelAdi || "Operatör yok"}
                        {" · "}
                        {formatDateTime(row.olusturmaTarihi)}
                      </div>
                    </td>
                    <td className="px-3 py-2">
                      {processing
                        ? "—"
                        : formatPeriod(row.donem?.baslangic, row.donem?.bitis)}
                    </td>
                    <td className="px-3 py-2">{processing ? "—" : "—"}</td>
                    <td className="px-3 py-2">
                      <StatusBadge
                        label={statusLabel(row.durum)}
                        tone={statusTone(row.durum)}
                      />
                    </td>
                    {canWrite ? (
                      <td className="px-3 py-2">
                        <button
                          type="button"
                          className="btn btn-danger"
                          aria-label="Sözleşmeyi sil"
                          onClick={(e) => void onDelete(row.id, e)}
                        >
                          Sil
                        </button>
                      </td>
                    ) : null}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      ) : null}
    </div>
  );
}
