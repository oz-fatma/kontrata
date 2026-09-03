"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "@/lib/auth";
import { LoadingState } from "./states";

export function AppShell({ children }: { children: React.ReactNode }) {
  const { ready, user, org, canViewMembers, logout } = useAuth();
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
  return (
    <div className="min-h-screen">
      <header className="border-b-[0.5px] border-[var(--line)]">
        <div className="mx-auto flex max-w-[1400px] items-center justify-between px-6 py-2">
          <Link href="/" className="text-[15px] font-medium">
            Kontrata
          </Link>
          <div className="flex items-center gap-3">
            <nav className="flex items-center gap-3 text-[13px]">
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
                      active ? "font-medium" : "text-[var(--muted)] hover:text-[var(--fg)]"
                    }
                  >
                    {item.label}
                  </Link>
                );
              })}
            </nav>
            <span className="cursor-default text-[12px] text-[var(--muted)]">
              {org?.ad ?? user.eposta}
            </span>
            <button type="button" className="btn" onClick={() => void logout()}>
              Çıkış
            </button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-[1400px] px-6 py-5">{children}</main>
    </div>
  );
}

export function AuthLayout({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-screen items-center justify-center px-6">
      <div className="w-full max-w-sm rounded-card border-[0.5px] border-[var(--line)] p-5">
        <h1 className="mb-4 text-[15px] font-medium">{title}</h1>
        {children}
      </div>
    </div>
  );
}
