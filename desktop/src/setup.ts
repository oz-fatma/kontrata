const form = document.getElementById("form") as HTMLFormElement;
const hata = document.getElementById("hata") as HTMLParagraphElement;
const kaydet = document.getElementById("kaydet") as HTMLButtonElement;
const mongo = document.getElementById("mongo") as HTMLInputElement;
const llmUrl = document.getElementById("llmUrl") as HTMLInputElement;
const llmToken = document.getElementById("llmToken") as HTMLInputElement;

form.addEventListener("submit", (ev) => {
  ev.preventDefault();
  void submit();
});

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
  });
  if (!result.ok) {
    hata.textContent = result.error;
    kaydet.disabled = false;
  }
}
