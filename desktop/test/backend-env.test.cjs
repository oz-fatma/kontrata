const test = require("node:test");
const assert = require("node:assert/strict");
const { mailerForSettings, mailerEnvVars } = require("../out/mailer-env");

const base = {
  mongoUri: "mongodb://localhost:27017",
  llmEndpointUrl: "",
  llmToken: "",
  jwtSecret: "x".repeat(32),
};

test("MAILER=console when smtp fields empty", () => {
  assert.equal(mailerForSettings(base), "console");
  const env = mailerEnvVars(base);
  assert.equal(env.MAILER, "console");
  assert.equal(env.SMTP_HOST, undefined);
  assert.equal(env.SMTP_FROM, undefined);
});

test("MAILER=smtp when host and from provided", () => {
  const settings = {
    ...base,
    smtpHost: "smtp.ornek.test",
    smtpPort: 465,
    smtpUser: "smtp-user",
    smtpPassword: "smtp-secret",
    smtpFrom: "kontrata@ornek.test",
  };
  assert.equal(mailerForSettings(settings), "smtp");
  const env = mailerEnvVars(settings);
  assert.equal(env.MAILER, "smtp");
  assert.equal(env.SMTP_HOST, "smtp.ornek.test");
  assert.equal(env.SMTP_PORT, "465");
  assert.equal(env.SMTP_USER, "smtp-user");
  assert.equal(env.SMTP_PASSWORD, "smtp-secret");
  assert.equal(env.SMTP_FROM, "kontrata@ornek.test");
});

test("MAILER=console when smtp incomplete", () => {
  const settings = { ...base, smtpHost: "smtp.ornek.test" };
  assert.equal(mailerForSettings(settings), "console");
  assert.equal(mailerEnvVars(settings).MAILER, "console");
});
