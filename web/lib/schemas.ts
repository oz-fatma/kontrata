import { z } from "zod";
import { HesapTipi } from "@/lib/enums";

const email = z
  .string()
  .trim()
  .min(1, "E-posta gerekli")
  .email("E-posta adresi geçersiz");

const password = z
  .string()
  .min(12, "Şifre en az 12 karakter olmalı");

export const registerSchema = z
  .object({
    eposta: email,
    sifre: password,
    sifreTekrar: z.string(),
    hesapTipi: z.enum(["BIREYSEL", "KURUMSAL"]),
    organizasyonAdi: z.string().trim().optional(),
  })
  .refine((v) => v.sifre === v.sifreTekrar, {
    message: "Şifreler eşleşmiyor",
    path: ["sifreTekrar"],
  })
  .refine(
    (v) => v.hesapTipi !== HesapTipi.Kurumsal || Boolean(v.organizasyonAdi?.trim()),
    {
      message: "Organizasyon adı gerekli",
      path: ["organizasyonAdi"],
    },
  );

export const loginSchema = z.object({
  eposta: email,
  sifre: z.string().min(1, "Şifre gerekli"),
});

export const mfaSchema = z.object({
  kod: z.string().regex(/^\d{6}$/, "6 haneli kod girin"),
});

export const resetRequestSchema = z.object({
  eposta: email,
});

export const resetPasswordSchema = z
  .object({
    yeniSifre: password,
    yeniSifreTekrar: z.string(),
  })
  .refine((v) => v.yeniSifre === v.yeniSifreTekrar, {
    message: "Şifreler eşleşmiyor",
    path: ["yeniSifreTekrar"],
  });

export const inviteSchema = z.object({
  eposta: email,
  rol: z.enum(["GORUNTULEYICI", "YONETICI"]),
});

export const deviceNameSchema = z.object({
  ad: z.string().trim().min(1, "Ad gerekli").max(80, "Ad çok uzun"),
});

export const deleteAccountSchema = z.object({
  token: z.string().trim().min(1, "Onay kodu gerekli"),
});

export type RegisterValues = z.infer<typeof registerSchema>;
export type LoginValues = z.infer<typeof loginSchema>;
export type MfaValues = z.infer<typeof mfaSchema>;
export type ResetRequestValues = z.infer<typeof resetRequestSchema>;
export type ResetPasswordValues = z.infer<typeof resetPasswordSchema>;
export type InviteValues = z.infer<typeof inviteSchema>;
