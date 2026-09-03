"use client";

import { useCallback, useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  UyeCikarDocument,
  UyeDavetEtDocument,
  UyeRolDegistirDocument,
  UyelerDocument,
} from "@/generated/graphql";
import { Rol } from "@/lib/enums";
import type { Member } from "@/lib/graphql-types";
import { useAuth } from "@/lib/auth";
import { AuthExpiredError, gqlRequest, graphqlMessage } from "@/lib/client";
import { roleLabel } from "@/lib/format";
import { inviteSchema, type InviteValues } from "@/lib/schemas";
import { AppShell } from "@/components/shell";
import { EmptyState, ErrorState, Field, LoadingState } from "@/components/states";

export default function MembersPage() {
  return (
    <AppShell>
      <MembersBody />
    </AppShell>
  );
}

function MembersBody() {
  const { org, userId, canManageMembers, canViewMembers } = useAuth();
  const [members, setMembers] = useState<Member[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const data = await gqlRequest(UyelerDocument);
      setMembers(data.uyeler);
    } catch (err) {
      if (err instanceof AuthExpiredError) {
        return;
      }
      setError(graphqlMessage(err));
      setMembers([]);
    }
  }, []);

  useEffect(() => {
    if (canViewMembers) {
      void load();
    }
  }, [canViewMembers, load]);

  if (!org || !canViewMembers) {
    return (
      <EmptyState
        title="Üye yönetimi yok"
        detail="Üyeler yalnızca kurumsal hesaplarda görünür."
      />
    );
  }

  return (
    <div className="flex flex-col gap-[var(--space-section)]">
      <div>
        <h1>Üyeler</h1>
        <p className="meta-text mt-1 tabular-nums">
          {org.ad}
          {members ? ` · ${members.length} kişi` : ""}
        </p>
      </div>

      {error ? <ErrorState message={error} onRetry={() => void load()} /> : null}

      {canManageMembers ? <InviteForm onInvited={() => void load()} /> : null}

      {members === null ? <LoadingState /> : null}
      {members && members.length === 0 && !error ? (
        <EmptyState title="Üye yok" />
      ) : null}
      {members && members.length > 0 ? (
        <ul className="card list-panel divide-y divide-[var(--border)] overflow-hidden">
          {members.map((member) => (
            <li key={member.id} className="flex flex-wrap items-center justify-between gap-2 text-[14px]">
              <div>
                <p className="font-medium text-[var(--ink)]">{member.eposta}</p>
                <p className="meta-text">{roleLabel(member.rol)}</p>
              </div>
              {canManageMembers && member.id !== userId && member.rol !== Rol.Sahip ? (
                <div className="flex flex-wrap items-center gap-2">
                  <label htmlFor={`rol-${member.id}`} className="sr-only">
                    Rol
                  </label>
                  <select
                    id={`rol-${member.id}`}
                    value={member.rol}
                    onChange={async (e) => {
                      try {
                        await gqlRequest(UyeRolDegistirDocument, {
                          kullaniciId: member.id,
                          rol: e.target.value as Member["rol"],
                        });
                        await load();
                      } catch (err) {
                        setError(graphqlMessage(err));
                      }
                    }}
                  >
                    <option value={Rol.Yonetici}>Yönetici</option>
                    <option value={Rol.Goruntuleyici}>Görüntüleyici</option>
                  </select>
                  <button
                    type="button"
                    className="btn btn-danger"
                    onClick={async () => {
                      if (!window.confirm("Üyeyi çıkarmak istiyor musunuz?")) {
                        return;
                      }
                      try {
                        await gqlRequest(UyeCikarDocument, { kullaniciId: member.id });
                        await load();
                      } catch (err) {
                        setError(graphqlMessage(err));
                      }
                    }}
                  >
                    Çıkar
                  </button>
                </div>
              ) : null}
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function InviteForm({ onInvited }: { onInvited: () => void }) {
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const form = useForm<InviteValues>({
    resolver: zodResolver(inviteSchema),
    defaultValues: { eposta: "", rol: Rol.Goruntuleyici },
  });

  return (
    <form
      className="card mb-[var(--space-card-gap)] flex flex-col gap-3 p-[var(--space-card)] sm:flex-row sm:items-end"
      onSubmit={form.handleSubmit(async (values) => {
        setError(null);
        setMessage(null);
        try {
          await gqlRequest(UyeDavetEtDocument, values);
          setMessage("Davet gönderildi (e-posta kayıtlıysa).");
          form.reset({ eposta: "", rol: values.rol });
          onInvited();
        } catch (err) {
          setError(graphqlMessage(err));
        }
      })}
    >
      <Field id="davet-eposta" label="E-posta" error={form.formState.errors.eposta?.message}>
        <input id="davet-eposta" type="email" autoComplete="email" {...form.register("eposta")} />
      </Field>
      <Field id="davet-rol" label="Rol">
        <select id="davet-rol" {...form.register("rol")}>
          <option value={Rol.Yonetici}>Yönetici</option>
          <option value={Rol.Goruntuleyici}>Görüntüleyici</option>
        </select>
      </Field>
      <button type="submit" className="btn btn-primary" disabled={form.formState.isSubmitting}>
        Davet et
      </button>
      {error ? <p className="w-full text-[12px] text-[var(--red)]">{error}</p> : null}
      {message ? <p className="w-full text-[12px] text-[var(--green)]">{message}</p> : null}
    </form>
  );
}
