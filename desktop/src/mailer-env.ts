import type { AppSettings } from "./config";
import { smtpConfigured } from "./config";

export function mailerForSettings(settings: AppSettings): "console" | "smtp" {
  if (smtpConfigured(settings)) {
    return "smtp";
  }
  return "console";
}

/** Arka plan sürecine geçirilecek MAILER ve SMTP ortam değişkenleri. */
export function mailerEnvVars(settings: AppSettings): Record<string, string> {
  const mailer = mailerForSettings(settings);
  const env: Record<string, string> = { MAILER: mailer };
  if (mailer !== "smtp") {
    return env;
  }
  env.SMTP_HOST = settings.smtpHost!.trim();
  env.SMTP_PORT = String(settings.smtpPort ?? 587);
  env.SMTP_FROM = settings.smtpFrom!.trim();
  if (settings.smtpUser?.trim()) {
    env.SMTP_USER = settings.smtpUser.trim();
  }
  if (settings.smtpPassword) {
    env.SMTP_PASSWORD = settings.smtpPassword;
  }
  return env;
}
