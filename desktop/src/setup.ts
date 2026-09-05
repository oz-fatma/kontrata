const form = document.getElementById("form") as HTMLFormElement;
const hata = document.getElementById("hata") as HTMLParagraphElement;
const kaydet = document.getElementById("kaydet") as HTMLButtonElement;
const mongo = document.getElementById("mongo") as HTMLInputElement;
const llmUrl = document.getElementById("llmUrl") as HTMLInputElement;
const llmToken = document.getElementById("llmToken") as HTMLInputElement;
const smtpHost = document.getElementById("smtpHost") as HTMLInputElement;
const smtpPort = document.getElementById("smtpPort") as HTMLInputElement;
const smtpUser = document.getElementById("smtpUser") as HTMLInputElement;
const smtpPassword = document.getElementById("smtpPassword") as HTMLInputElement;
const smtpFrom = document.getElementById("smtpFrom") as HTMLInputElement;

form.addEventListener("submit", (ev) => {
  ev.preventDefault();
  void submit();
});

void loadExisting();

async function loadExisting(): Promise<void> {
  const api = window.kontrata;
  if (!api?.loadSettings) {
    return;
  }
  const saved = await api.loadSettings();
  if (!saved) {
    return;
  }
  mongo.value = saved.mongoUri;
  llmUrl.value = saved.llmEndpointUrl;
  smtpHost.value = saved.smtpHost;
  smtpPort.value = saved.smtpPort;
  smtpUser.value = saved.smtpUser;
  smtpFrom.value = saved.smtpFrom;
}

async function submit(): Promise<void> {
  hata.textContent = "";
  kaydet.disabled = true;
  const api = window.kontrata;
  if (!api?.saveSettings) {
    hata.textContent = "kurulum arayüzü yüklenemedi";
    kaydet.disabled = false;
    return;
  }
  const result = await api.saveSettings({
    mongoUri: mongo.value,
    llmEndpointUrl: llmUrl.value,
    llmToken: llmToken.value,
    smtpHost: smtpHost.value,
    smtpPort: smtpPort.value,
    smtpUser: smtpUser.value,
    smtpPassword: smtpPassword.value,
    smtpFrom: smtpFrom.value,
  });
  if (!result.ok) {
    hata.textContent = result.error;
    kaydet.disabled = false;
  }
}
