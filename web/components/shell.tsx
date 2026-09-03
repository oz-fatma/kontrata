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
  const nav = [
    { href: "/", label: "Sözleşmeler" },
    { href: "/ayarlar", label: "Ayarlar" },
  ];
  if (canViewMembers) {
    nav.push({ href: "/ayarlar/uyeler", label: "Üyeler" });
  }
  if (isOwner && org) {
    nav.push({ href: "/ayarlar/yonetici", label: "Yönetici" });
  }
  return (
    <div className="min-h-screen">
      <header className="border-b-[0.5px] border-[var(--border)] bg-[var(--surface)]">
        <div className="mx-auto flex max-w-[1400px] items-center justify-between px-6 py-3">
          <BrandLogo />
          <div className="flex items-center gap-4">
            <nav className="flex items-center gap-4 text-[14px]">
              {nav.map((item) => {
                const active =
                  item.href === "/"
                    ? path === "/"
                    : path === item.href || path.startsWith(`${item.href}/`);
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
            <span className="meta-text cursor-default">{org?.ad ?? user.eposta}</span>
            <button type="button" className="btn" onClick={() => void logout()}>
              Çıkış
            </button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-[1400px] px-6 py-[var(--space-section)]">{children}</main>
    </div>
  );
}

export function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center px-6 py-10">
      <div className="card w-full max-w-sm p-[var(--space-card)]">
        <div className="mb-6 flex justify-center">
          <BrandLogo href="/giris/" />
        </div>
        {children}
      </div>
    </div>
  );
}
