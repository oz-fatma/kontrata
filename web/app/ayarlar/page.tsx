"use client";

import { useCallback, useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  CihazAdlandirDocument,
  CihazGuvenilirYapDocument,
  CihazKaldirDocument,
  CihazlarimDocument,
  HesapSilDocument,
  HesapSilmeIsteDocument,
  OturumlarimDocument,
  TumOturumlariKapatDocument,
  type CihazlarimQuery,
  type OturumlarimQuery,
} from "@/generated/graphql";
import { useAuth } from "@/lib/auth";
import { AuthExpiredError, gqlRequest, graphqlMessage } from "@/lib/client";
import { formatDateTime, formatUserAgent } from "@/lib/format";
import { deleteAccountSchema } from "@/lib/schemas";
import { AppShell } from "@/components/shell";
import { EmptyState, ErrorState, Field, LoadingState } from "@/components/states";

type Device = CihazlarimQuery["cihazlarim"][number];
type Session = OturumlarimQuery["oturumlarim"][number];

export default function SettingsPage() {
  return (
    <AppShell>
      <SettingsBody />
    </AppShell>
  );
}

function SettingsBody() {
  const { org, user, logout, isOwner } = useAuth();
  const [devices, setDevices] = useState<Device[] | null>(null);
  const [sessions, setSessions] = useState<Session[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const [d, s] = await Promise.all([
        gqlRequest(CihazlarimDocument),
        gqlRequest(OturumlarimDocument),
      ]);
      setDevices(d.cihazlarim);
      setSessions(s.oturumlarim);
    } catch (err) {
      if (err instanceof AuthExpiredError) {
        return;
      }
      setError(graphqlMessage(err));
      setDevices([]);
      setSessions([]);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className="flex flex-col gap-[var(--space-section)]">
      <div>
        <h1>Ayarlar</h1>
        <p className="meta-text mt-1">
          {org?.ad ?? "Bireysel hesap"}
          {user?.eposta ? ` · ${user.eposta}` : ""}
        </p>
      </div>

      {error ? <ErrorState message={error} onRetry={() => void load()} /> : null}

      <section>
        <h2 className="mb-3">Cihazlar</h2>
        {devices === null ? <LoadingState /> : null}
        {devices && devices.length === 0 && !error ? (
          <EmptyState title="Kayıtlı cihaz yok" detail="Giriş yapılan cihazlar burada görünür." />
        ) : null}
        {devices && devices.length > 0 ? (
          <ul className="card list-panel divide-y divide-[var(--border)] overflow-hidden">
            {devices.map((device) => (
              <DeviceRow key={device.id} device={device} onChanged={() => void load()} />
            ))}
          </ul>
        ) : null}
      </section>

      <section>
        <h2 className="mb-3">Oturumlar</h2>
        {sessions === null ? <LoadingState /> : null}
        {sessions && sessions.length === 0 && !error ? (
          <EmptyState title="Açık oturum yok" />
        ) : null}
        {sessions && sessions.length > 0 ? (
          <div>
            <ul className="card list-panel divide-y divide-[var(--border)] overflow-hidden">
              {sessions.map((session) => (
                <li key={session.id} className="text-[14px]">
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-medium text-[var(--ink)]">
                      {formatUserAgent(session.kullaniciAjani)}
                    </span>
                    {session.mevcutMu ? (
                      <span className="meta-text text-[var(--blue)]">Bu oturum</span>
                    ) : null}
                  </div>
                  <p className="meta-text tabular-nums">
                    Son erişim: {formatDateTime(session.olusturmaTarihi)}
                    {" · IP: "}
                    {session.ipAdresi || "—"}
                  </p>
                </li>
              ))}
            </ul>
            <button
              type="button"
              className="btn mt-3"
              onClick={async () => {
                try {
                  await gqlRequest(TumOturumlariKapatDocument);
                  await logout();
                } catch (err) {
                  setError(graphqlMessage(err));
                }
              }}
            >
              Tüm oturumları kapat
            </button>
          </div>
        ) : null}
      </section>

      {isOwner ? <DeleteAccount /> : null}
    </div>
  );
}

function DeviceRow({
  device,
  onChanged,
}: {
  device: Device;
  onChanged: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(device.ad);
  const [err, setErr] = useState<string | null>(null);

  async function saveName() {
    setErr(null);
    try {
      await gqlRequest(CihazAdlandirDocument, { id: device.id, ad: name.trim() });
      setEditing(false);
      onChanged();
    } catch (e) {
      setErr(graphqlMessage(e));
    }
  }

  return (
    <li className="text-[14px]">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          {editing ? (
            <div className="flex gap-2">
              <label htmlFor={`cihaz-${device.id}`} className="sr-only">
                Cihaz adı
              </label>
              <input
                id={`cihaz-${device.id}`}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
              <button type="button" className="btn btn-primary" onClick={() => void saveName()}>
                Kaydet
              </button>
            </div>
          ) : (
            <p className="font-medium">
              {device.ad || "Adsız cihaz"}
              {device.guvenilir ? (
                <span className="ml-2 meta-text font-normal text-[var(--green)]">Güvenilir</span>
              ) : null}
            </p>
          )}
          <p className="meta-text tabular-nums">
            Son görülme {formatDateTime(device.sonGorulme)}
            {device.ipAdresi ? ` · ${device.ipAdresi}` : ""}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button type="button" className="btn" onClick={() => setEditing((v) => !v)}>
            {editing ? "Vazgeç" : "Adlandır"}
          </button>
          {!device.guvenilir ? (
            <button
              type="button"
              className="btn"
              onClick={async () => {
                try {
                  await gqlRequest(CihazGuvenilirYapDocument, { id: device.id });
                  onChanged();
                } catch (e) {
                  setErr(graphqlMessage(e));
                }
              }}
            >
              Güvenilir yap
            </button>
          ) : null}
          <button
            type="button"
            className="btn btn-danger"
            onClick={async () => {
              if (!window.confirm("Bu cihazı kaldırmak istiyor musunuz?")) {
                return;
              }
              try {
                await gqlRequest(CihazKaldirDocument, { id: device.id });
                onChanged();
              } catch (e) {
                setErr(graphqlMessage(e));
              }
            }}
          >
            Kaldır
          </button>
        </div>
      </div>
      {err ? <p className="mt-1 text-[12px] text-[var(--red)]">{err}</p> : null}
    </li>
  );
}

function DeleteAccount() {
  const { logout } = useAuth();
  const [requested, setRequested] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const form = useForm<{ token: string }>({
    resolver: zodResolver(deleteAccountSchema),
    defaultValues: { token: "" },
  });

  if (done) {
    return (
      <section>
        <h2>Hesabı sil</h2>
        <p className="mt-2 text-[13px]">Hesabınız silindi.</p>
      </section>
    );
  }

  return (
    <section>
      <h2>Hesabı sil</h2>
      <p className="mt-1 text-[14px] text-[var(--ink-muted)]">
        Silme kalıcıdır. Kurumsal hesapta başka üye varsa önce devir veya organizasyonu
        silmeniz gerekir.
      </p>
      {error ? <ErrorState message={error} /> : null}
      {!requested ? (
        <button
          type="button"
          className="btn btn-danger mt-2"
          onClick={async () => {
            setError(null);
            try {
              await gqlRequest(HesapSilmeIsteDocument);
              setRequested(true);
            } catch (err) {
              setError(graphqlMessage(err));
            }
          }}
        >
          Silme kodu gönder
        </button>
      ) : (
        <form
          className="mt-3 flex max-w-sm flex-col gap-2"
          onSubmit={form.handleSubmit(async (values) => {
            setError(null);
            try {
              await gqlRequest(HesapSilDocument, { token: values.token });
              setDone(true);
              await logout();
            } catch (err) {
              setError(graphqlMessage(err));
            }
          })}
        >
          <p className="text-[13px]">E-postanıza gelen onay kodunu girin.</p>
          <Field id="sil-token" label="Onay kodu" error={form.formState.errors.token?.message}>
            <input id="sil-token" autoComplete="one-time-code" {...form.register("token")} />
          </Field>
          <button type="submit" className="btn btn-danger" disabled={form.formState.isSubmitting}>
            Hesabı kalıcı olarak sil
          </button>
        </form>
      )}
    </section>
  );
}
