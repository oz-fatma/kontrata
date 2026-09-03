"use client";

import Link from "next/link";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { EpostaDogrulaDocument } from "@/generated/graphql";
import { gqlRequest, graphqlMessage } from "@/lib/client";
import { verifyTokenSchema, type VerifyTokenValues } from "@/lib/schemas";
import { parseVerifyInput, verifyFailedMessage } from "@/lib/verify-token";
import { ErrorState, Field } from "@/components/states";

export function VerifyTokenForm({
  hint,
  onVerified,
}: {
  hint?: string;
  onVerified?: () => void;
}) {
  const [serverError, setServerError] = useState<string | null>(null);
  const [ok, setOk] = useState(false);
  const form = useForm<VerifyTokenValues>({
    resolver: zodResolver(verifyTokenSchema),
    defaultValues: { token: "" },
  });

  async function onSubmit(values: VerifyTokenValues) {
    const token = parseVerifyInput(values.token);
    if (!token) {
      form.setError("token", { message: "Doğrulama kodu gerekli" });
      return;
    }
    setServerError(null);
    try {
      const data = await gqlRequest(EpostaDogrulaDocument, { token });
      if (!data.epostaDogrula) {
        setServerError(verifyFailedMessage);
        return;
      }
      setOk(true);
      onVerified?.();
    } catch (err) {
      setServerError(graphqlMessage(err));
    }
  }

  if (ok) {
    return (
      <div>
        <p className="text-[13px]">E-postanız doğrulandı. Giriş yapabilirsiniz.</p>
        <Link href="/giris/" className="btn btn-primary mt-4 inline-flex w-full justify-center">
          Giriş yap
        </Link>
      </div>
    );
  }

  return (
    <form className="flex flex-col gap-3" onSubmit={form.handleSubmit(onSubmit)}>
      {hint ? <p className="text-[13px] text-[var(--muted)]">{hint}</p> : null}
      {serverError ? <ErrorState message={serverError} /> : null}
      <Field
        id="dogrulama-token"
        label="Doğrulama kodu"
        error={form.formState.errors.token?.message}
      >
        <input
          id="dogrulama-token"
          autoComplete="one-time-code"
          spellCheck={false}
          {...form.register("token")}
        />
      </Field>
      <button
        type="submit"
        className="btn btn-primary w-full"
        disabled={form.formState.isSubmitting}
      >
        {form.formState.isSubmitting ? "Doğrulanıyor…" : "Doğrula"}
      </button>
    </form>
  );
}
