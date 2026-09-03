"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { KayitOlDocument } from "@/generated/graphql";
import { gqlRequest, graphqlMessage } from "@/lib/client";
import { HesapTipi } from "@/lib/enums";
import { registerSchema, type RegisterValues } from "@/lib/schemas";
import { AuthLayout } from "@/components/shell";
import { ErrorState, Field } from "@/components/states";

export default function RegisterPage() {
  const router = useRouter();
  const [serverError, setServerError] = useState<string | null>(null);
  const [done, setDone] = useState<string | null>(null);
  const form = useForm<RegisterValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: {
      eposta: "",
      sifre: "",
      sifreTekrar: "",
      hesapTipi: HesapTipi.Bireysel,
      organizasyonAdi: "",
    },
  });
  const hesapTipi = form.watch("hesapTipi");

  async function onSubmit(values: RegisterValues) {
    setServerError(null);
    try {
      const data = await gqlRequest(KayitOlDocument, {
        eposta: values.eposta,
        sifre: values.sifre,
        hesapTipi: values.hesapTipi,
        organizasyonAdi:
          values.hesapTipi === HesapTipi.Kurumsal
            ? values.organizasyonAdi
            : undefined,
      });
      setDone(data.kayitOl.mesaj);
    } catch (err) {
      setServerError(graphqlMessage(err));
    }
  }

  return (
    <AuthLayout title="Kayıt ol">
      {done ? (
        <div>
          <p className="text-[13px]">{done}</p>
          <button
            type="button"
            className="btn btn-primary mt-4 w-full"
            onClick={() => router.push("/giris/")}
          >
            Girişe git
          </button>
        </div>
      ) : (
        <form className="flex flex-col gap-3" onSubmit={form.handleSubmit(onSubmit)}>
          {serverError ? <ErrorState message={serverError} /> : null}
          <Field id="eposta" label="E-posta" error={form.formState.errors.eposta?.message}>
            <input id="eposta" type="email" autoComplete="email" {...form.register("eposta")} />
          </Field>
          <Field id="sifre" label="Şifre" error={form.formState.errors.sifre?.message}>
            <input
              id="sifre"
              type="password"
              autoComplete="new-password"
              {...form.register("sifre")}
            />
          </Field>
          <Field
            id="sifreTekrar"
            label="Şifre tekrar"
            error={form.formState.errors.sifreTekrar?.message}
          >
            <input
              id="sifreTekrar"
              type="password"
              autoComplete="new-password"
              {...form.register("sifreTekrar")}
            />
          </Field>
          <Field id="hesapTipi" label="Hesap tipi">
            <select id="hesapTipi" {...form.register("hesapTipi")}>
              <option value={HesapTipi.Bireysel}>Bireysel</option>
              <option value={HesapTipi.Kurumsal}>Kurumsal</option>
            </select>
          </Field>
          {hesapTipi === HesapTipi.Kurumsal ? (
            <Field
              id="organizasyonAdi"
              label="Organizasyon adı"
              error={form.formState.errors.organizasyonAdi?.message}
            >
              <input id="organizasyonAdi" type="text" {...form.register("organizasyonAdi")} />
            </Field>
          ) : null}
          <button
            type="submit"
            className="btn btn-primary w-full"
            disabled={form.formState.isSubmitting}
          >
            {form.formState.isSubmitting ? "Kaydediliyor…" : "Hesap oluştur"}
          </button>
          <p className="text-[12px] text-[var(--muted)]">
            Hesabınız var mı?{" "}
            <Link href="/giris/" className="underline">
              Giriş yapın
            </Link>
          </p>
        </form>
      )}
    </AuthLayout>
  );
}
