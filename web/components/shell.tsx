"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "@/lib/auth";
import { BrandLogo } from "./brand-logo";
import { LoadingState } from "./states";

export function AppShell({ children }: { children: React.ReactNode }) {
  const { ready, user, org, canViewMembers, isOwner, logout } = useAuth();
  const pathname = usePathname();
  if (!ready) {
    return (
      <div className="mx-auto max-w-[1400px] px-6">
        <LoadingState />
      </div>
    );
  }
  if (!user) {
    return (
      <div className="mx-auto max-w-[1400px] px-6">
        <LoadingState label="Girişe yönlendiriliyor" />
      </div>
    );
  }
  const path = pathname.replace(/\/$/, "") || "/";
  const inSettings = path === "/ayarlar" || path.startsWith("/ayarlar/");
  const orgLabel = org?.ad ?? user.eposta;
  const nav = [
    { href: "/", label: "Sözleşmeler" },
    { href: "/ayarlar", label: "Ayarlar" },
  ];
  const settingsNav = [
    { href: "/ayarlar", label: "Genel", exact: true },
    ...(canViewMembers
      ? [{ href: "/ayarlar/uyeler", label: "Üyeler", exact: false }]
      : []),
    ...(isOwner && org
      ? [{ href: "/ayarlar/yonetici", label: "Yönetici", exact: false }]
      : []),
  ];
  return (
    <div className="min-h-screen">
      <header className="border-b-[0.5px] border-[var(--border)] bg-[var(--surface)]">
        <div className="mx-auto flex max-w-[1400px] flex-wrap items-center justify-between gap-x-4 gap-y-2 px-6 py-3">
          <BrandLogo />
          <div className="flex min-w-0 flex-wrap items-center justify-end gap-2 sm:gap-3">
            <nav className="flex items-center gap-3 text-[14px] sm:gap-4">
              {nav.map((item) => {
                const active =
                  item.href === "/"
                    ? path === "/" || path.startsWith("/sozlesme")
                    : inSettings;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={
                      active
                        ? "font-semibold text-[var(--accent)]"
                        : "font-medium text-[var(--ink-muted)] hover:text-[var(--ink)]"
                    }
                  >
                    {item.label}
                  </Link>
                );
              })}
            </nav>
            <span
              className="meta-text hidden max-w-[10rem] truncate cursor-default sm:inline md:max-w-[14rem]"
              title={orgLabel}
            >
              {orgLabel}
            </span>
            <button type="button" className="btn" onClick={() => void logout()}>
              Çıkış
            </button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-[1400px] px-6 py-[var(--space-section)]">
        {inSettings && settingsNav.length > 1 ? (
          <nav
            className="mb-[var(--space-card-gap)] flex flex-wrap gap-1 border-b-[0.5px] border-[var(--border)] pb-0"
            aria-label="Ayarlar"
          >
            {settingsNav.map((item) => {
              const active = item.exact
                ? path === item.href
                : path === item.href || path.startsWith(`${item.href}/`);
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`-mb-px border-b-2 px-3 py-2 text-[13px] font-medium transition-colors ${
                    active
                      ? "border-[var(--accent)] text-[var(--accent)]"
                      : "border-transparent text-[var(--ink-muted)] hover:text-[var(--ink)]"
                  }`}
                >
                  {item.label}
                </Link>
              );
            })}
          </nav>
        ) : null}
        {children}
      </main>
    </div>
  );
}

export function AuthLayout({
  children,
  title,
}: {
  children: React.ReactNode;
  title?: string;
}) {
  return (
    <div className="flex min-h-screen items-center justify-center px-6 py-10">
      <div className="card w-full max-w-sm p-[var(--space-card)]">
        <div className="mb-6 flex justify-center">
          <BrandLogo href="/giris/" />
        </div>
        {title ? <h1 className="mb-4 text-center text-[18px]">{title}</h1> : null}
        {children}
      </div>
    </div>
  );
}
