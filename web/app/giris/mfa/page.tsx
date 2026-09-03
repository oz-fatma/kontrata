"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { GirisYapDocument, MfaDogrulaDocument } from "@/generated/graphql";
import { useAuth } from "@/lib/auth";
import { gqlRequest, graphqlMessage } from "@/lib/client";
import { mfaSchema, type MfaValues } from "@/lib/schemas";
import { MFA_TTL_SEC, getPendingPassword, setPendingPassword } from "@/lib/session";
import { AuthLayout } from "@/components/shell";
import { ErrorState, Field } from "@/components/states";

export default function MfaPage() {
  const router = useRouter();
  const { setTokens } = useAuth();
  const [serverError, setServerError] = useState<string | null>(null);
  const [left, setLeft] = useState(MFA_TTL_SEC);
  const [token, setToken] = useState<string | null>(null);
  const form = useForm<MfaValues>({
    resolver: zodResolver(mfaSchema),
    defaultValues: { kod: "" },
  });

  useEffect(() => {
    const t = sessionStorage.getItem("kontrata.mfa.token");
    const started = Number(sessionStorage.getItem("kontrata.mfa.started") ?? "0");
    if (!t) {
      router.replace("/giris/");
      return;
    }
    setToken(t);
    const tick = () => {
      const elapsed = Math.floor((Date.now() - started) / 1000);
      setLeft(Math.max(0, MFA_TTL_SEC - elapsed));
    };
    tick();
    const id = window.setInterval(tick, 500);
    return () => window.clearInterval(id);
  }, [router]);

  async function onSubmit(values: MfaValues) {
    if (!token) {
      return;
    }
    setServerError(null);
    try {
      const data = await gqlRequest(MfaDogrulaDocument, { geciciToken: token, kod: values.kod });
      sessionStorage.removeItem("kontrata.mfa.token");
      sessionStorage.removeItem("kontrata.mfa.email");
      sessionStorage.removeItem("kontrata.mfa.started");
      setPendingPassword(null);
      await setTokens(data.mfaDogrula.erisimJetonu, data.mfaDogrula.yenilemeJetonu);
      router.replace("/");
    } catch (err) {
      setServerError(graphqlMessage(err));
    }
  }

  async function resend() {
    const email = sessionStorage.getItem("kontrata.mfa.email");
    const password = getPendingPassword();
    if (!email || !password) {
      setServerError("Oturum bilgisi kayboldu. Tekrar giriş yapın.");
      return;
    }
    setServerError(null);
    try {
      const data = await gqlRequest(GirisYapDocument, { eposta: email, sifre: password });
      sessionStorage.setItem("kontrata.mfa.token", data.girisYap.geciciToken);
      sessionStorage.setItem("kontrata.mfa.started", String(Date.now()));
      setToken(data.girisYap.geciciToken);
      setLeft(MFA_TTL_SEC);
    } catch (err) {
      setServerError(graphqlMessage(err));
    }
  }

  const mm = String(Math.floor(left / 60)).padStart(2, "0");
  const ss = String(left % 60).padStart(2, "0");

  return (
    <AuthLayout title="Doğrulama kodu">
      <form className="flex flex-col gap-3" onSubmit={form.handleSubmit(onSubmit)}>
        {serverError ? <ErrorState message={serverError} /> : null}
        <p className="text-[13px] text-[var(--muted)]">
          E-postanıza gelen 6 haneli kodu girin. Kalan süre {mm}:{ss}.
        </p>
        <Field id="kod" label="Doğrulama kodu" error={form.formState.errors.kod?.message}>
          <input
            id="kod"
            inputMode="numeric"
            autoComplete="one-time-code"
            maxLength={6}
            {...form.register("kod")}
          />
        </Field>
        <button
          type="submit"
          className="btn btn-primary w-full"
          disabled={form.formState.isSubmitting || left === 0}
        >
          {form.formState.isSubmitting ? "Doğrulanıyor…" : "Doğrula"}
        </button>
        <button type="button" className="btn w-full" onClick={() => void resend()}>
          Tekrar gönder
        </button>
        <p className="text-[12px] text-[var(--muted)]">
          <Link href="/giris/" className="underline">
            Girişe dön
          </Link>
        </p>
      </form>
    </AuthLayout>
  );
}
