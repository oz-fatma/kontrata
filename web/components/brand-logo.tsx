import Link from "next/link";

export function BrandLogo({ href = "/" }: { href?: string }) {
  return (
    <Link
      href={href}
      className="inline-flex items-center gap-2.5 text-[15px] font-semibold tracking-[-0.01em] text-[var(--ink)]"
    >
      <span
        className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-[7px] bg-[var(--accent)] text-[15px] font-bold leading-none text-white"
        aria-hidden
      >
        K
      </span>
      Kontrata
    </Link>
  );
}
