"use client";

import Link from "next/link";
import { Suspense, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { SifreSifirlaDocument, SifreSifirlamaIsteDocument } from "@/generated/graphql";
import { gqlRequest, graphqlMessage } from "@/lib/client";
import {
  resetPasswordSchema,
  resetRequestSchema,
  type ResetPasswordValues,
  type ResetRequestValues,
} from "@/lib/schemas";
import { AuthLayout } from "@/components/shell";
import { ErrorState, Field, LoadingState } from "@/components/states";

export default function ResetPage() {
  return (
    <Suspense fallback={<LoadingState />}>
      <ResetInner />
    </Suspense>
  );
}

function ResetInner() {
  const params = useSearchParams();
  const token = params.get("token") ?? "";
  return token ? <ResetConfirm token={token} /> : <ResetRequest />;
}

function ResetRequest() {
  const [serverError, setServerError] = useState<string | null>(null);
  const [done, setDone] = useState(false);
  const form = useForm<ResetRequestValues>({
    resolver: zodResolver(resetRequestSchema),
    defaultValues: { eposta: "" },
  });

  async function onSubmit(values: ResetRequestValues) {
    setServerError(null);
    try {
      await gqlRequest(SifreSifirlamaIsteDocument, values);
      setDone(true);
    } catch (err) {
      setServerError(graphqlMessage(err));
    }
  }

  return (
    <AuthLayout>
      {done ? (
        <p className="text-[13px]">
          E-posta kayıtlıysa sıfırlama bağlantısı gönderildi.
        </p>
      ) : (
        <form className="flex flex-col gap-3" onSubmit={form.handleSubmit(onSubmit)}>
          {serverError ? <ErrorState message={serverError} /> : null}
          <Field id="eposta" label="E-posta" error={form.formState.errors.eposta?.message}>
            <input id="eposta" type="email" autoComplete="email" {...form.register("eposta")} />
          </Field>
          <button
            type="submit"
            className="btn btn-primary w-full"
            disabled={form.formState.isSubmitting}
          >
            {form.formState.isSubmitting ? "Gönderiliyor…" : "Bağlantı gönder"}
          </button>
        </form>
      )}
      <p className="mt-3 text-[12px] text-[var(--muted)]">
        <Link href="/giris/" className="underline">
          Girişe dön
        </Link>
      </p>
    </AuthLayout>
  );
}

function ResetConfirm({ token }: { token: string }) {
  const [serverError, setServerError] = useState<string | null>(null);
  const [done, setDone] = useState(false);
  const form = useForm<ResetPasswordValues>({
    resolver: zodResolver(resetPasswordSchema),
    defaultValues: { yeniSifre: "", yeniSifreTekrar: "" },
  });

  async function onSubmit(values: ResetPasswordValues) {
    setServerError(null);
    try {
      await gqlRequest(SifreSifirlaDocument, {
        token,
        yeniSifre: values.yeniSifre,
      });
      setDone(true);
    } catch (err) {
      setServerError(graphqlMessage(err));
    }
  }

  return (
    <AuthLayout>
      {done ? (
        <div>
          <p className="text-[13px]">Şifreniz güncellendi.</p>
          <Link href="/giris/" className="btn btn-primary mt-4 inline-flex">
            Giriş yap
          </Link>
        </div>
      ) : (
        <form className="flex flex-col gap-3" onSubmit={form.handleSubmit(onSubmit)}>
          {serverError ? <ErrorState message={serverError} /> : null}
          <Field
            id="yeniSifre"
            label="Yeni şifre"
            error={form.formState.errors.yeniSifre?.message}
          >
            <input
              id="yeniSifre"
              type="password"
              autoComplete="new-password"
              {...form.register("yeniSifre")}
            />
          </Field>
          <Field
            id="yeniSifreTekrar"
            label="Yeni şifre tekrar"
            error={form.formState.errors.yeniSifreTekrar?.message}
          >
            <input
              id="yeniSifreTekrar"
              type="password"
              autoComplete="new-password"
              {...form.register("yeniSifreTekrar")}
            />
          </Field>
          <button
            type="submit"
            className="btn btn-primary w-full"
            disabled={form.formState.isSubmitting}
          >
            {form.formState.isSubmitting ? "Kaydediliyor…" : "Şifreyi güncelle"}
          </button>
        </form>
      )}
    </AuthLayout>
  );
}
