import type { Metadata } from "next";
import { AuthProvider } from "@/lib/auth";
import "./globals.css";

export const metadata: Metadata = {
  title: "Kontrata",
  description: "Kontenjan sözleşmesi çıkarımı",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="tr">
      <body>
        <AuthProvider>{children}</AuthProvider>
      </body>
    </html>
  );
}
