"use client";

import Link from "next/link";
import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { EpostaDogrulaDocument } from "@/generated/graphql";
import { gqlRequest, graphqlMessage } from "@/lib/client";
import { verifyFailedMessage } from "@/lib/verify-token";
import { AuthLayout } from "@/components/shell";
import { ErrorState, LoadingState } from "@/components/states";
import { VerifyTokenForm } from "@/components/verify-token-form";

export default function VerifyPage() {
  return (
    <Suspense fallback={<LoadingState />}>
      <VerifyInner />
    </Suspense>
  );
}

function VerifyInner() {
  const params = useSearchParams();
  const token = params.get("token") ?? "";
  const [state, setState] = useState<"idle" | "loading" | "ok" | "error">(
    token ? "loading" : "idle",
  );
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (!token) {
      setState("idle");
      return;
    }
    let cancelled = false;
    setState("loading");
    (async () => {
      try {
        const data = await gqlRequest(EpostaDogrulaDocument, { token });
        if (!cancelled) {
          if (data.epostaDogrula) {
            setState("ok");
          } else {
            setState("error");
            setMessage(verifyFailedMessage);
          }
        }
      } catch (err) {
        if (!cancelled) {
          setState("error");
          setMessage(graphqlMessage(err));
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [token]);

  return (
    <AuthLayout>
      {state === "loading" ? <LoadingState label="Doğrulanıyor" /> : null}
      {state === "ok" ? (
        <div>
          <p className="text-[13px]">E-postanız doğrulandı. Giriş yapabilirsiniz.</p>
          <Link href="/giris/" className="btn btn-primary mt-4 inline-flex">
            Giriş yap
          </Link>
        </div>
      ) : null}
      {state === "idle" ? (
        <VerifyTokenForm hint="E-postadaki doğrulama kodunu yapıştırın." />
      ) : null}
      {state === "error" ? (
        <div className="flex flex-col gap-4">
          <ErrorState message={message} />
          <VerifyTokenForm hint="Kodu elle yapıştırıp yeniden deneyin." />
        </div>
      ) : null}
    </AuthLayout>
  );
}
