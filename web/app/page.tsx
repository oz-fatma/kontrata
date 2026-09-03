"use client";

import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
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
  isExtractPending,
  listFindingCell,
  statusLabel,
  statusTone,
} from "@/lib/format";
import { usePolling } from "@/lib/use-polling";
import { AppShell } from "@/components/shell";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { EmptyState, ErrorState, LoadingState } from "@/components/states";
import { Spinner } from "@/components/spinner";
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
  const [pendingDelete, setPendingDelete] = useState<Row | null>(null);

  const load = useCallback(
    async (opts?: { silent?: boolean }) => {
      if (!opts?.silent) {
        setError(null);
      }
      try {
        const data = await gqlRequest(SozlesmelerDocument, {
          limit: 100,
          offset: 0,
        });
        setRows(data.sozlesmeler ?? []);
      } catch (err) {
        if (err instanceof AuthExpiredError) {
          router.replace("/giris/");
          return;
        }
        if (!opts?.silent) {
          setError(graphqlMessage(err));
        }
      }
    },
    [router],
  );

  useEffect(() => {
    void load();
  }, [load]);

  usePolling(() => load({ silent: true }), true);

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

  const onOpen = useCallback(
    (id: string) => {
      router.push(`/sozlesme/?id=${encodeURIComponent(id)}`);
    },
    [router],
  );

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

  const onDelete = useCallback((id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const row = rows?.find((r) => r.id === id) ?? null;
    setPendingDelete(row);
  }, [rows]);

  async function confirmDelete() {
    if (!pendingDelete) {
      return;
    }
    const id = pendingDelete.id;
    setBusy(true);
    setError(null);
    try {
      await gqlRequest(SozlesmeSilDocument, { id });
      setPendingDelete(null);
      await load();
    } catch (err) {
      setError(graphqlMessage(err));
    } finally {
      setBusy(false);
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
              {filtered.map((row) => (
                <ContractRow
                  key={`${row.id}:${row.durum}:${row.guncellemeTarihi}`}
                  row={row}
                  canWrite={canWrite}
                  onOpen={onOpen}
                  onDelete={onDelete}
                />
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
      {pendingDelete ? (
        <ConfirmDialog
          title="Sözleşmeyi sil"
          message="Bu sözleşme ve yüklenen dosya kalıcı olarak silinecek"
          confirmLabel="Sil"
          busy={busy}
          onCancel={() => setPendingDelete(null)}
          onConfirm={() => void confirmDelete()}
        />
      ) : null}
    </div>
  );
}

const ContractRow = memo(function ContractRow({
  row,
  canWrite,
  onOpen,
  onDelete,
}: {
  row: Row;
  canWrite: boolean;
  onOpen: (id: string) => void;
  onDelete: (id: string, e: React.MouseEvent) => void;
}) {
  const processing = isExtractPending(row.durum);
  return (
    <tr
      className="cursor-pointer border-b-[0.5px] border-[var(--line)] last:border-0 hover:bg-[var(--subtle)]"
      onClick={() => onOpen(row.id)}
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
        {processing ? "—" : formatPeriod(row.donem?.baslangic, row.donem?.bitis)}
      </td>
      <td className="px-3 py-2">
        {processing ? (
          "—"
        ) : (
          <FindingCount findings={row.bulgular} />
        )}
      </td>
      <td className="px-3 py-2">
        <span className="inline-flex items-center gap-1.5">
          {processing ? <Spinner /> : null}
          <StatusBadge label={statusLabel(row.durum)} tone={statusTone(row.durum)} />
        </span>
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
});

function FindingCount({ findings }: { findings: Row["bulgular"] }) {
  const cell = listFindingCell(findings);
  const color =
    cell.tone === "red"
      ? "var(--red)"
      : cell.tone === "yellow"
        ? "var(--yellow)"
        : "var(--muted)";
  return <span style={{ color }}>{cell.text}</span>;
}
