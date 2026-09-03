import { FileCheck2 } from "lucide-react";
import Link from "next/link";

export function BrandLogo({ href = "/" }: { href?: string }) {
  return (
    <Link
      href={href}
      className="inline-flex items-center gap-2 text-[15px] font-semibold tracking-[-0.01em] text-[var(--ink)]"
    >
      <FileCheck2
        className="h-[18px] w-[18px] shrink-0 text-[var(--accent)]"
        strokeWidth={2}
        aria-hidden
      />
      Kontrata
    </Link>
  );
}
