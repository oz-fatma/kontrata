import { Suspense } from "react";
import { LoadingState } from "@/components/states";
import ContractDetailPage from "./contract-detail";

export default function Page() {
  return (
    <Suspense fallback={<LoadingState />}>
      <ContractDetailPage />
    </Suspense>
  );
}
