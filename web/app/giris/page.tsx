"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { Suspense, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { GirisYapDocument } from "@/generated/graphql";
import { gqlRequest, graphqlMessage } from "@/lib/client";
import { loginSchema, type LoginValues } from "@/lib/schemas";
import { setPendingPassword } from "@/lib/session";
import { AuthLayout } from "@/components/shell";
import { ErrorState, Field, LoadingState } from "@/components/states";

export default function LoginPage() {
  return (
    <Suspense fallback={<LoadingState />}>
      <LoginForm />
    </Suspense>
  );
}

function LoginForm() {
  const router = useRouter();
  const [serverError, setServerError] = useState<string | null>(null);
  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { eposta: "", sifre: "" },
  });

  async function onSubmit(values: LoginValues) {
    setServerError(null);
    try {
      const data = await gqlRequest(GirisYapDocument, values);
      sessionStorage.setItem("kontrata.mfa.token", data.girisYap.geciciToken);
      sessionStorage.setItem("kontrata.mfa.email", values.eposta);
      sessionStorage.setItem("kontrata.mfa.started", String(Date.now()));
      setPendingPassword(values.sifre);
      router.push("/giris/mfa/");
    } catch (err) {
      setServerError(graphqlMessage(err));
    }
  }

  return (
    <AuthLayout title="Giriş yap">
      <form className="flex flex-col gap-3" onSubmit={form.handleSubmit(onSubmit)}>
        {serverError ? <ErrorState message={serverError} /> : null}
        <Field id="eposta" label="E-posta" error={form.formState.errors.eposta?.message}>
          <input id="eposta" type="email" autoComplete="email" {...form.register("eposta")} />
        </Field>
        <Field id="sifre" label="Şifre" error={form.formState.errors.sifre?.message}>
          <input
            id="sifre"
            type="password"
            autoComplete="current-password"
            {...form.register("sifre")}
          />
        </Field>
        <button
          type="submit"
          className="btn btn-primary w-full"
          disabled={form.formState.isSubmitting}
        >
          {form.formState.isSubmitting ? "Gönderiliyor…" : "Devam et"}
        </button>
        <p className="text-[12px] text-[var(--muted)]">
          <Link href="/sifre-sifirla/" className="underline">
            Şifremi unuttum
          </Link>
          {" · "}
          <Link href="/kayit/" className="underline">
            Kayıt ol
          </Link>
        </p>
      </form>
    </AuthLayout>
  );
}
