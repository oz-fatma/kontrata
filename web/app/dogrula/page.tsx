"use client";

import Link from "next/link";
import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { EpostaDogrulaDocument } from "@/generated/graphql";
import { gqlRequest, graphqlMessage } from "@/lib/client";
import { AuthLayout } from "@/components/shell";
import { ErrorState, LoadingState } from "@/components/states";

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
  const [state, setState] = useState<"loading" | "ok" | "error">("loading");
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (!token) {
      setState("error");
      setMessage("Doğrulama kodu eksik.");
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        await gqlRequest(EpostaDogrulaDocument, { token });
        if (!cancelled) {
          setState("ok");
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
    <AuthLayout title="E-posta doğrulama">
      {state === "loading" ? <LoadingState label="Doğrulanıyor" /> : null}
      {state === "ok" ? (
        <div>
          <p className="text-[13px]">E-postanız doğrulandı. Giriş yapabilirsiniz.</p>
          <Link href="/giris/" className="btn btn-primary mt-4 inline-flex">
            Giriş yap
          </Link>
        </div>
      ) : null}
      {state === "error" ? (
        <div>
          <ErrorState message={message} />
          <Link href="/giris/" className="btn mt-3 inline-flex">
            Girişe dön
          </Link>
        </div>
      ) : null}
    </AuthLayout>
  );
}
