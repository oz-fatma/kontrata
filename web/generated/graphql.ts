/* eslint-disable */
/** Internal type. DO NOT USE DIRECTLY. */
type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
/** Internal type. DO NOT USE DIRECTLY. */
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
import { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';
export type AltDonemGirdi = {
  ad: string;
  baslangic: string;
  bitis: string;
};

export type BildirimYontemi =
  | 'BELIRTILMEMIS'
  | 'EPOSTA'
  | 'FAKS'
  | 'SISTEM'
  | 'YAZILI';

export type BulguKaynagi =
  | 'KURAL'
  | 'MODEL';

export type BulguOnemi =
  | 'BILGI'
  | 'KRITIK'
  | 'UYARI';

export type CikarimMetaGirdi = {
  alanYolu: string;
  elleDuzeltildi?: boolean | null | undefined;
  guven?: number | null | undefined;
  kaynakMadde?: string | null | undefined;
  kaynakSayfa?: number | null | undefined;
};

export type CocukPolitikasiGirdi = {
  indirimYuzde?: number | null | undefined;
  kosul?: string | null | undefined;
  ucretsiz?: boolean | null | undefined;
  yasMax?: number | null | undefined;
  yasMin?: number | null | undefined;
};

export type DonemGirdi = {
  altDonemler?: Array<AltDonemGirdi> | null | undefined;
  baslangic?: string | null | undefined;
  bitis?: string | null | undefined;
};

export type FiyatBirimi =
  | 'KISI_GECELIK'
  | 'ODA_GECELIK';

export type FiyatGirdi = {
  altDonemAd?: string | null | undefined;
  birim: FiyatBirimi;
  odaTipi: string;
  pansiyon?: Pansiyon | null | undefined;
  tutar: number;
};

export type HesapTipi =
  | 'BIREYSEL'
  | 'KURUMSAL';

export type IptalKosuluGirdi = {
  gun?: number | null | undefined;
  kapsam?: string | null | undefined;
  tazminatAciklama?: string | null | undefined;
};

export type KurEsasi =
  | 'BELIRTILMEMIS'
  | 'CIKIS_GUNU_TCMB'
  | 'GIRIS_GUNU_TCMB'
  | 'SABIT_KUR';

export type NoShowGirdi = {
  sorumluTaraf?: string | null | undefined;
  tazminatAciklama?: string | null | undefined;
};

export type OdaKontenjaniGirdi = {
  aciklama?: string | null | undefined;
  adet: number;
  odaTipi: string;
};

export type OdemeGirdi = {
  avansAciklama?: string | null | undefined;
  avansVar?: boolean | null | undefined;
  faturaSonrasiGun?: number | null | undefined;
};

export type OrganizasyonDurumu =
  | 'AKTIF'
  | 'ASKIDA';

export type OverbookingGirdi = {
  aciklama?: string | null | undefined;
  sorumluTaraf?: string | null | undefined;
};

export type Pansiyon =
  | 'AI'
  | 'BB'
  | 'BELIRTILMEMIS'
  | 'FB'
  | 'HB'
  | 'RO';

export type PromptTipi =
  | 'DENETCI'
  | 'OKUYUCU';

export type ReleaseKapsami =
  | 'BELIRTILMEMIS'
  | 'HER_IKISI'
  | 'ISIM_LISTESI'
  | 'KONTENJAN_IADESI';

export type ReleaseKuraliGirdi = {
  gun: number;
  kapsam?: ReleaseKapsami | null | undefined;
  kaynakIfade?: string | null | undefined;
};

export type Rol =
  | 'GORUNTULEYICI'
  | 'SAHIP'
  | 'YONETICI';

export type Sezon =
  | 'BELIRTILMEMIS'
  | 'KIS'
  | 'YAZ'
  | 'YILLIK';

export type SozlesmeDurumu =
  | 'HATA'
  | 'INCELENMEYI_BEKLIYOR'
  | 'ISLENIYOR'
  | 'ONAYLANDI'
  | 'YUKLENDI';

export type SozlesmeGirdi = {
  cikarimMeta?: Array<CikarimMetaGirdi> | null | undefined;
  cocukPolitikasi?: Array<CocukPolitikasiGirdi> | null | undefined;
  donem?: DonemGirdi | null | undefined;
  dosyaAdi?: string | null | undefined;
  durum?: SozlesmeDurumu | null | undefined;
  fiyatlar?: Array<FiyatGirdi> | null | undefined;
  iptalKosullari?: Array<IptalKosuluGirdi> | null | undefined;
  meta?: SozlesmeMetaGirdi | null | undefined;
  noShow?: NoShowGirdi | null | undefined;
  odaKontenjanlari?: Array<OdaKontenjaniGirdi> | null | undefined;
  odeme?: OdemeGirdi | null | undefined;
  overbooking?: OverbookingGirdi | null | undefined;
  release?: ReleaseKuraliGirdi | null | undefined;
  stopSale?: Array<StopSaleAraligiGirdi> | null | undefined;
};

export type SozlesmeMetaGirdi = {
  acenteAdi?: string | null | undefined;
  imzaTarihi?: string | null | undefined;
  kurEsasi?: KurEsasi | null | undefined;
  otelAdi?: string | null | undefined;
  paraBirimi?: string | null | undefined;
  sezon?: Sezon | null | undefined;
  sozlesmeTipi?: SozlesmeTipi | null | undefined;
  yetkiliMahkeme?: string | null | undefined;
};

export type SozlesmeTipi =
  | 'BELIRTILMEMIS'
  | 'BLOK_REZERVASYON'
  | 'BLOK_SATIN_ALMA'
  | 'GARANTISIZ'
  | 'ISTEGE_BAGLI'
  | 'KISMEN_GARANTILI'
  | 'SERBEST_SATIS'
  | 'TAMAMEN_GARANTILI';

export type StopSaleAraligiGirdi = {
  baslangic?: string | null | undefined;
  bildirimYontemi?: BildirimYontemi | null | undefined;
  bitis?: string | null | undefined;
  kapsam?: string | null | undefined;
  kaynakIfade?: string | null | undefined;
};

export type SozlesmelerQueryVariables = Exact<{
  limit?: number | null | undefined;
  offset?: number | null | undefined;
}>;


export type SozlesmelerQuery = { sozlesmeler: Array<{ id: string, dosyaAdi: string | null, durum: SozlesmeDurumu, olusturmaTarihi: string, guncellemeTarihi: string, meta: { otelAdi: string | null, acenteAdi: string | null } | null, donem: { baslangic: string | null, bitis: string | null } | null, bulgular: Array<{ onem: BulguOnemi }> }> };

export type SozlesmeQueryVariables = Exact<{
  id: string | number;
}>;


export type SozlesmeQuery = { sozlesme: { id: string, dosyaAdi: string | null, durum: SozlesmeDurumu, olusturmaTarihi: string, guncellemeTarihi: string, duzeltmeler: Array<string> | null, semaHatalari: Array<string> | null, islemSuresi: number | null, denetciSuresi: number | null, promptSurumu: number | null, meta: { otelAdi: string | null, acenteAdi: string | null, sozlesmeTipi: SozlesmeTipi | null, sezon: Sezon | null, paraBirimi: string | null, kurEsasi: KurEsasi | null, yetkiliMahkeme: string | null, imzaTarihi: string | null } | null, donem: { baslangic: string | null, bitis: string | null, altDonemler: Array<{ ad: string, baslangic: string, bitis: string }> | null } | null, odaKontenjanlari: Array<{ odaTipi: string, adet: number, aciklama: string | null }> | null, fiyatlar: Array<{ odaTipi: string, pansiyon: Pansiyon | null, tutar: number, birim: FiyatBirimi, altDonemAd: string | null }> | null, release: { gun: number, kapsam: ReleaseKapsami | null, kaynakIfade: string | null } | null, stopSale: Array<{ baslangic: string | null, bitis: string | null, kapsam: string | null, bildirimYontemi: BildirimYontemi | null, kaynakIfade: string | null }> | null, cikarimMeta: Array<{ alanYolu: string, guven: number | null, kaynakSayfa: number | null, kaynakMadde: string | null, elleDuzeltildi: boolean | null }> | null, bulgular: Array<{ kod: string, baslik: string, aciklama: string, onem: BulguOnemi, kaynak: BulguKaynagi, alanYolu: string | null }> } | null };

export type OturumlarimQueryVariables = Exact<{ [key: string]: never; }>;


export type OturumlarimQuery = { oturumlarim: Array<{ id: string, olusturmaTarihi: string, sonKullanma: string, ipAdresi: string | null, kullaniciAjani: string | null, mevcutMu: boolean }> };

export type CihazlarimQueryVariables = Exact<{ [key: string]: never; }>;


export type CihazlarimQuery = { cihazlarim: Array<{ id: string, ad: string, guvenilir: boolean, ilkGorulme: string, sonGorulme: string, ipAdresi: string | null, kullaniciAjani: string | null }> };

export type OrganizasyonumQueryVariables = Exact<{ [key: string]: never; }>;


export type OrganizasyonumQuery = { organizasyonum: { id: string, ad: string, vergiNo: string | null, durum: OrganizasyonDurumu, olusturmaTarihi: string } | null };

export type UyelerQueryVariables = Exact<{ [key: string]: never; }>;


export type UyelerQuery = { uyeler: Array<{ id: string, eposta: string, rol: Rol, hesapTipi: HesapTipi }> };

export type KayitOlMutationVariables = Exact<{
  eposta: string;
  sifre: string;
  hesapTipi?: HesapTipi | null | undefined;
  organizasyonAdi?: string | null | undefined;
}>;


export type KayitOlMutation = { kayitOl: { basarili: boolean, mesaj: string } };

export type EpostaDogrulaMutationVariables = Exact<{
  token: string;
}>;


export type EpostaDogrulaMutation = { epostaDogrula: boolean };

export type DogrulamaTekrarGonderMutationVariables = Exact<{
  eposta: string;
}>;


export type DogrulamaTekrarGonderMutation = { dogrulamaTekrarGonder: boolean };

export type SifreSifirlamaIsteMutationVariables = Exact<{
  eposta: string;
}>;


export type SifreSifirlamaIsteMutation = { sifreSifirlamaIste: boolean };

export type SifreSifirlaMutationVariables = Exact<{
  token: string;
  yeniSifre: string;
}>;


export type SifreSifirlaMutation = { sifreSifirla: boolean };

export type GirisYapMutationVariables = Exact<{
  eposta: string;
  sifre: string;
}>;


export type GirisYapMutation = { girisYap: { mfaGerekli: boolean, geciciToken: string } };

export type MfaDogrulaMutationVariables = Exact<{
  geciciToken: string;
  kod: string;
}>;


export type MfaDogrulaMutation = { mfaDogrula: { erisimJetonu: string, yenilemeJetonu: string } };

export type JetonYenileMutationVariables = Exact<{
  yenilemeJetonu: string;
}>;


export type JetonYenileMutation = { jetonYenile: { erisimJetonu: string, yenilemeJetonu: string } };

export type CikisYapMutationVariables = Exact<{ [key: string]: never; }>;


export type CikisYapMutation = { cikisYap: boolean };

export type TumOturumlariKapatMutationVariables = Exact<{ [key: string]: never; }>;


export type TumOturumlariKapatMutation = { tumOturumlariKapat: number };

export type CihazAdlandirMutationVariables = Exact<{
  id: string | number;
  ad: string;
}>;


export type CihazAdlandirMutation = { cihazAdlandir: { id: string, ad: string } };

export type CihazKaldirMutationVariables = Exact<{
  id: string | number;
}>;


export type CihazKaldirMutation = { cihazKaldir: boolean };

export type CihazGuvenilirYapMutationVariables = Exact<{
  id: string | number;
}>;


export type CihazGuvenilirYapMutation = { cihazGuvenilirYap: { id: string, guvenilir: boolean } };

export type HesapSilmeIsteMutationVariables = Exact<{ [key: string]: never; }>;


export type HesapSilmeIsteMutation = { hesapSilmeIste: boolean };

export type HesapSilMutationVariables = Exact<{
  token: string;
}>;


export type HesapSilMutation = { hesapSil: boolean };

export type SozlesmeYukleMutationVariables = Exact<{
  dosya: File;
}>;


export type SozlesmeYukleMutation = { sozlesmeYukle: { id: string, dosyaAdi: string | null, durum: SozlesmeDurumu } };

export type SozlesmeSilMutationVariables = Exact<{
  id: string | number;
}>;


export type SozlesmeSilMutation = { sozlesmeSil: boolean };

export type SozlesmeOnaylaMutationVariables = Exact<{
  id: string | number;
}>;


export type SozlesmeOnaylaMutation = { sozlesmeOnayla: { id: string, durum: SozlesmeDurumu } };

export type SozlesmeAlanGuncelleMutationVariables = Exact<{
  id: string | number;
  alanYolu: string;
  deger: unknown;
}>;


export type SozlesmeAlanGuncelleMutation = { sozlesmeAlanGuncelle: { id: string, durum: SozlesmeDurumu, guncellemeTarihi: string, denetciSuresi: number | null, meta: { otelAdi: string | null, acenteAdi: string | null, sozlesmeTipi: SozlesmeTipi | null, sezon: Sezon | null, paraBirimi: string | null, kurEsasi: KurEsasi | null, yetkiliMahkeme: string | null, imzaTarihi: string | null } | null, donem: { baslangic: string | null, bitis: string | null, altDonemler: Array<{ ad: string, baslangic: string, bitis: string }> | null } | null, odaKontenjanlari: Array<{ odaTipi: string, adet: number, aciklama: string | null }> | null, fiyatlar: Array<{ odaTipi: string, pansiyon: Pansiyon | null, tutar: number, birim: FiyatBirimi, altDonemAd: string | null }> | null, release: { gun: number, kapsam: ReleaseKapsami | null, kaynakIfade: string | null } | null, stopSale: Array<{ baslangic: string | null, bitis: string | null, kapsam: string | null, bildirimYontemi: BildirimYontemi | null, kaynakIfade: string | null }> | null, cikarimMeta: Array<{ alanYolu: string, guven: number | null, kaynakSayfa: number | null, kaynakMadde: string | null, elleDuzeltildi: boolean | null }> | null, bulgular: Array<{ kod: string, baslik: string, aciklama: string, onem: BulguOnemi, kaynak: BulguKaynagi, alanYolu: string | null }> } };

export type SozlesmeGuncelleMutationVariables = Exact<{
  id: string | number;
  girdi: SozlesmeGirdi;
}>;


export type SozlesmeGuncelleMutation = { sozlesmeGuncelle: { id: string, durum: SozlesmeDurumu } };

export type UyeDavetEtMutationVariables = Exact<{
  eposta: string;
  rol: Rol;
}>;


export type UyeDavetEtMutation = { uyeDavetEt: boolean };

export type UyeRolDegistirMutationVariables = Exact<{
  kullaniciId: string | number;
  rol: Rol;
}>;


export type UyeRolDegistirMutation = { uyeRolDegistir: { id: string, eposta: string, rol: Rol } };

export type UyeCikarMutationVariables = Exact<{
  kullaniciId: string | number;
}>;


export type UyeCikarMutation = { uyeCikar: boolean };

export type AktifPromptQueryVariables = Exact<{
  tip: PromptTipi;
}>;


export type AktifPromptQuery = { aktifPrompt: { id: string, tip: PromptTipi, icerik: string, surum: number, aktif: boolean, olusturmaTarihi: string, olusturanKullaniciId: string } };

export type PromptSurumleriQueryVariables = Exact<{
  tip: PromptTipi;
}>;


export type PromptSurumleriQuery = { promptSurumleri: Array<{ id: string, tip: PromptTipi, icerik: string, surum: number, aktif: boolean, olusturmaTarihi: string, olusturanKullaniciId: string }> };

export type AyarlarQueryVariables = Exact<{ [key: string]: never; }>;


export type AyarlarQuery = { ayarlar: { denetciRiskEsigi: number, maxToken: number, guncellemeTarihi: string, guncelleyenKullaniciId: string | null } };

export type PromptGuncelleMutationVariables = Exact<{
  tip: PromptTipi;
  icerik: string;
}>;


export type PromptGuncelleMutation = { promptGuncelle: { id: string, tip: PromptTipi, icerik: string, surum: number, aktif: boolean, olusturmaTarihi: string, olusturanKullaniciId: string } };

export type PromptSurumeDonMutationVariables = Exact<{
  id: string | number;
}>;


export type PromptSurumeDonMutation = { promptSurumeDon: { id: string, tip: PromptTipi, icerik: string, surum: number, aktif: boolean, olusturmaTarihi: string, olusturanKullaniciId: string } };

export type AyarlariGuncelleMutationVariables = Exact<{
  denetciRiskEsigi?: number | null | undefined;
  maxToken?: number | null | undefined;
}>;


export type AyarlariGuncelleMutation = { ayarlariGuncelle: { denetciRiskEsigi: number, maxToken: number, guncellemeTarihi: string, guncelleyenKullaniciId: string | null } };


export const SozlesmelerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Sozlesmeler"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"offset"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sozlesmeler"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}},{"kind":"Argument","name":{"kind":"Name","value":"offset"},"value":{"kind":"Variable","name":{"kind":"Name","value":"offset"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"dosyaAdi"}},{"kind":"Field","name":{"kind":"Name","value":"durum"}},{"kind":"Field","name":{"kind":"Name","value":"olusturmaTarihi"}},{"kind":"Field","name":{"kind":"Name","value":"guncellemeTarihi"}},{"kind":"Field","name":{"kind":"Name","value":"meta"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"otelAdi"}},{"kind":"Field","name":{"kind":"Name","value":"acenteAdi"}}]}},{"kind":"Field","name":{"kind":"Name","value":"donem"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"baslangic"}},{"kind":"Field","name":{"kind":"Name","value":"bitis"}}]}},{"kind":"Field","name":{"kind":"Name","value":"bulgular"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"onem"}}]}}]}}]}}]} as unknown as DocumentNode<SozlesmelerQuery, SozlesmelerQueryVariables>;
export const SozlesmeDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Sozlesme"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sozlesme"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"dosyaAdi"}},{"kind":"Field","name":{"kind":"Name","value":"durum"}},{"kind":"Field","name":{"kind":"Name","value":"olusturmaTarihi"}},{"kind":"Field","name":{"kind":"Name","value":"guncellemeTarihi"}},{"kind":"Field","name":{"kind":"Name","value":"meta"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"otelAdi"}},{"kind":"Field","name":{"kind":"Name","value":"acenteAdi"}},{"kind":"Field","name":{"kind":"Name","value":"sozlesmeTipi"}},{"kind":"Field","name":{"kind":"Name","value":"sezon"}},{"kind":"Field","name":{"kind":"Name","value":"paraBirimi"}},{"kind":"Field","name":{"kind":"Name","value":"kurEsasi"}},{"kind":"Field","name":{"kind":"Name","value":"yetkiliMahkeme"}},{"kind":"Field","name":{"kind":"Name","value":"imzaTarihi"}}]}},{"kind":"Field","name":{"kind":"Name","value":"donem"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"baslangic"}},{"kind":"Field","name":{"kind":"Name","value":"bitis"}},{"kind":"Field","name":{"kind":"Name","value":"altDonemler"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ad"}},{"kind":"Field","name":{"kind":"Name","value":"baslangic"}},{"kind":"Field","name":{"kind":"Name","value":"bitis"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"odaKontenjanlari"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"odaTipi"}},{"kind":"Field","name":{"kind":"Name","value":"adet"}},{"kind":"Field","name":{"kind":"Name","value":"aciklama"}}]}},{"kind":"Field","name":{"kind":"Name","value":"fiyatlar"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"odaTipi"}},{"kind":"Field","name":{"kind":"Name","value":"pansiyon"}},{"kind":"Field","name":{"kind":"Name","value":"tutar"}},{"kind":"Field","name":{"kind":"Name","value":"birim"}},{"kind":"Field","name":{"kind":"Name","value":"altDonemAd"}}]}},{"kind":"Field","name":{"kind":"Name","value":"release"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"gun"}},{"kind":"Field","name":{"kind":"Name","value":"kapsam"}},{"kind":"Field","name":{"kind":"Name","value":"kaynakIfade"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stopSale"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"baslangic"}},{"kind":"Field","name":{"kind":"Name","value":"bitis"}},{"kind":"Field","name":{"kind":"Name","value":"kapsam"}},{"kind":"Field","name":{"kind":"Name","value":"bildirimYontemi"}},{"kind":"Field","name":{"kind":"Name","value":"kaynakIfade"}}]}},{"kind":"Field","name":{"kind":"Name","value":"cikarimMeta"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"alanYolu"}},{"kind":"Field","name":{"kind":"Name","value":"guven"}},{"kind":"Field","name":{"kind":"Name","value":"kaynakSayfa"}},{"kind":"Field","name":{"kind":"Name","value":"kaynakMadde"}},{"kind":"Field","name":{"kind":"Name","value":"elleDuzeltildi"}}]}},{"kind":"Field","name":{"kind":"Name","value":"duzeltmeler"}},{"kind":"Field","name":{"kind":"Name","value":"semaHatalari"}},{"kind":"Field","name":{"kind":"Name","value":"islemSuresi"}},{"kind":"Field","name":{"kind":"Name","value":"denetciSuresi"}},{"kind":"Field","name":{"kind":"Name","value":"promptSurumu"}},{"kind":"Field","name":{"kind":"Name","value":"bulgular"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"kod"}},{"kind":"Field","name":{"kind":"Name","value":"baslik"}},{"kind":"Field","name":{"kind":"Name","value":"aciklama"}},{"kind":"Field","name":{"kind":"Name","value":"onem"}},{"kind":"Field","name":{"kind":"Name","value":"kaynak"}},{"kind":"Field","name":{"kind":"Name","value":"alanYolu"}}]}}]}}]}}]} as unknown as DocumentNode<SozlesmeQuery, SozlesmeQueryVariables>;
export const OturumlarimDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Oturumlarim"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"oturumlarim"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"olusturmaTarihi"}},{"kind":"Field","name":{"kind":"Name","value":"sonKullanma"}},{"kind":"Field","name":{"kind":"Name","value":"ipAdresi"}},{"kind":"Field","name":{"kind":"Name","value":"kullaniciAjani"}},{"kind":"Field","name":{"kind":"Name","value":"mevcutMu"}}]}}]}}]} as unknown as DocumentNode<OturumlarimQuery, OturumlarimQueryVariables>;
export const CihazlarimDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Cihazlarim"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cihazlarim"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"ad"}},{"kind":"Field","name":{"kind":"Name","value":"guvenilir"}},{"kind":"Field","name":{"kind":"Name","value":"ilkGorulme"}},{"kind":"Field","name":{"kind":"Name","value":"sonGorulme"}},{"kind":"Field","name":{"kind":"Name","value":"ipAdresi"}},{"kind":"Field","name":{"kind":"Name","value":"kullaniciAjani"}}]}}]}}]} as unknown as DocumentNode<CihazlarimQuery, CihazlarimQueryVariables>;
export const OrganizasyonumDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Organizasyonum"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"organizasyonum"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"ad"}},{"kind":"Field","name":{"kind":"Name","value":"vergiNo"}},{"kind":"Field","name":{"kind":"Name","value":"durum"}},{"kind":"Field","name":{"kind":"Name","value":"olusturmaTarihi"}}]}}]}}]} as unknown as DocumentNode<OrganizasyonumQuery, OrganizasyonumQueryVariables>;
export const UyelerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Uyeler"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"uyeler"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"eposta"}},{"kind":"Field","name":{"kind":"Name","value":"rol"}},{"kind":"Field","name":{"kind":"Name","value":"hesapTipi"}}]}}]}}]} as unknown as DocumentNode<UyelerQuery, UyelerQueryVariables>;
export const KayitOlDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"KayitOl"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eposta"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"sifre"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"hesapTipi"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"HesapTipi"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"organizasyonAdi"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"kayitOl"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"eposta"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eposta"}}},{"kind":"Argument","name":{"kind":"Name","value":"sifre"},"value":{"kind":"Variable","name":{"kind":"Name","value":"sifre"}}},{"kind":"Argument","name":{"kind":"Name","value":"hesapTipi"},"value":{"kind":"Variable","name":{"kind":"Name","value":"hesapTipi"}}},{"kind":"Argument","name":{"kind":"Name","value":"organizasyonAdi"},"value":{"kind":"Variable","name":{"kind":"Name","value":"organizasyonAdi"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"basarili"}},{"kind":"Field","name":{"kind":"Name","value":"mesaj"}}]}}]}}]} as unknown as DocumentNode<KayitOlMutation, KayitOlMutationVariables>;
export const EpostaDogrulaDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"EpostaDogrula"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"token"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"epostaDogrula"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"token"},"value":{"kind":"Variable","name":{"kind":"Name","value":"token"}}}]}]}}]} as unknown as DocumentNode<EpostaDogrulaMutation, EpostaDogrulaMutationVariables>;
export const DogrulamaTekrarGonderDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DogrulamaTekrarGonder"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eposta"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"dogrulamaTekrarGonder"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"eposta"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eposta"}}}]}]}}]} as unknown as DocumentNode<DogrulamaTekrarGonderMutation, DogrulamaTekrarGonderMutationVariables>;
export const SifreSifirlamaIsteDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SifreSifirlamaIste"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eposta"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sifreSifirlamaIste"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"eposta"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eposta"}}}]}]}}]} as unknown as DocumentNode<SifreSifirlamaIsteMutation, SifreSifirlamaIsteMutationVariables>;
export const SifreSifirlaDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SifreSifirla"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"token"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"yeniSifre"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sifreSifirla"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"token"},"value":{"kind":"Variable","name":{"kind":"Name","value":"token"}}},{"kind":"Argument","name":{"kind":"Name","value":"yeniSifre"},"value":{"kind":"Variable","name":{"kind":"Name","value":"yeniSifre"}}}]}]}}]} as unknown as DocumentNode<SifreSifirlaMutation, SifreSifirlaMutationVariables>;
export const GirisYapDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"GirisYap"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eposta"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"sifre"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"girisYap"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"eposta"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eposta"}}},{"kind":"Argument","name":{"kind":"Name","value":"sifre"},"value":{"kind":"Variable","name":{"kind":"Name","value":"sifre"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"mfaGerekli"}},{"kind":"Field","name":{"kind":"Name","value":"geciciToken"}}]}}]}}]} as unknown as DocumentNode<GirisYapMutation, GirisYapMutationVariables>;
export const MfaDogrulaDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"MfaDogrula"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"geciciToken"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"kod"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"mfaDogrula"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"geciciToken"},"value":{"kind":"Variable","name":{"kind":"Name","value":"geciciToken"}}},{"kind":"Argument","name":{"kind":"Name","value":"kod"},"value":{"kind":"Variable","name":{"kind":"Name","value":"kod"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"erisimJetonu"}},{"kind":"Field","name":{"kind":"Name","value":"yenilemeJetonu"}}]}}]}}]} as unknown as DocumentNode<MfaDogrulaMutation, MfaDogrulaMutationVariables>;
export const JetonYenileDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"JetonYenile"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"yenilemeJetonu"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"jetonYenile"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"yenilemeJetonu"},"value":{"kind":"Variable","name":{"kind":"Name","value":"yenilemeJetonu"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"erisimJetonu"}},{"kind":"Field","name":{"kind":"Name","value":"yenilemeJetonu"}}]}}]}}]} as unknown as DocumentNode<JetonYenileMutation, JetonYenileMutationVariables>;
export const CikisYapDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CikisYap"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cikisYap"}}]}}]} as unknown as DocumentNode<CikisYapMutation, CikisYapMutationVariables>;
export const TumOturumlariKapatDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TumOturumlariKapat"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"tumOturumlariKapat"}}]}}]} as unknown as DocumentNode<TumOturumlariKapatMutation, TumOturumlariKapatMutationVariables>;
export const CihazAdlandirDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CihazAdlandir"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ad"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cihazAdlandir"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"ad"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ad"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"ad"}}]}}]}}]} as unknown as DocumentNode<CihazAdlandirMutation, CihazAdlandirMutationVariables>;
export const CihazKaldirDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CihazKaldir"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cihazKaldir"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}]}]}}]} as unknown as DocumentNode<CihazKaldirMutation, CihazKaldirMutationVariables>;
export const CihazGuvenilirYapDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CihazGuvenilirYap"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cihazGuvenilirYap"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"guvenilir"}}]}}]}}]} as unknown as DocumentNode<CihazGuvenilirYapMutation, CihazGuvenilirYapMutationVariables>;
export const HesapSilmeIsteDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"HesapSilmeIste"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hesapSilmeIste"}}]}}]} as unknown as DocumentNode<HesapSilmeIsteMutation, HesapSilmeIsteMutationVariables>;
export const HesapSilDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"HesapSil"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"token"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hesapSil"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"token"},"value":{"kind":"Variable","name":{"kind":"Name","value":"token"}}}]}]}}]} as unknown as DocumentNode<HesapSilMutation, HesapSilMutationVariables>;
export const SozlesmeYukleDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SozlesmeYukle"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"dosya"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Upload"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sozlesmeYukle"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"dosya"},"value":{"kind":"Variable","name":{"kind":"Name","value":"dosya"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"dosyaAdi"}},{"kind":"Field","name":{"kind":"Name","value":"durum"}}]}}]}}]} as unknown as DocumentNode<SozlesmeYukleMutation, SozlesmeYukleMutationVariables>;
export const SozlesmeSilDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SozlesmeSil"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sozlesmeSil"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}]}]}}]} as unknown as DocumentNode<SozlesmeSilMutation, SozlesmeSilMutationVariables>;
export const SozlesmeOnaylaDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SozlesmeOnayla"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sozlesmeOnayla"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"durum"}}]}}]}}]} as unknown as DocumentNode<SozlesmeOnaylaMutation, SozlesmeOnaylaMutationVariables>;
export const SozlesmeAlanGuncelleDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SozlesmeAlanGuncelle"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"alanYolu"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"deger"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"JSON"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sozlesmeAlanGuncelle"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"alanYolu"},"value":{"kind":"Variable","name":{"kind":"Name","value":"alanYolu"}}},{"kind":"Argument","name":{"kind":"Name","value":"deger"},"value":{"kind":"Variable","name":{"kind":"Name","value":"deger"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"durum"}},{"kind":"Field","name":{"kind":"Name","value":"guncellemeTarihi"}},{"kind":"Field","name":{"kind":"Name","value":"meta"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"otelAdi"}},{"kind":"Field","name":{"kind":"Name","value":"acenteAdi"}},{"kind":"Field","name":{"kind":"Name","value":"sozlesmeTipi"}},{"kind":"Field","name":{"kind":"Name","value":"sezon"}},{"kind":"Field","name":{"kind":"Name","value":"paraBirimi"}},{"kind":"Field","name":{"kind":"Name","value":"kurEsasi"}},{"kind":"Field","name":{"kind":"Name","value":"yetkiliMahkeme"}},{"kind":"Field","name":{"kind":"Name","value":"imzaTarihi"}}]}},{"kind":"Field","name":{"kind":"Name","value":"donem"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"baslangic"}},{"kind":"Field","name":{"kind":"Name","value":"bitis"}},{"kind":"Field","name":{"kind":"Name","value":"altDonemler"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ad"}},{"kind":"Field","name":{"kind":"Name","value":"baslangic"}},{"kind":"Field","name":{"kind":"Name","value":"bitis"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"odaKontenjanlari"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"odaTipi"}},{"kind":"Field","name":{"kind":"Name","value":"adet"}},{"kind":"Field","name":{"kind":"Name","value":"aciklama"}}]}},{"kind":"Field","name":{"kind":"Name","value":"fiyatlar"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"odaTipi"}},{"kind":"Field","name":{"kind":"Name","value":"pansiyon"}},{"kind":"Field","name":{"kind":"Name","value":"tutar"}},{"kind":"Field","name":{"kind":"Name","value":"birim"}},{"kind":"Field","name":{"kind":"Name","value":"altDonemAd"}}]}},{"kind":"Field","name":{"kind":"Name","value":"release"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"gun"}},{"kind":"Field","name":{"kind":"Name","value":"kapsam"}},{"kind":"Field","name":{"kind":"Name","value":"kaynakIfade"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stopSale"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"baslangic"}},{"kind":"Field","name":{"kind":"Name","value":"bitis"}},{"kind":"Field","name":{"kind":"Name","value":"kapsam"}},{"kind":"Field","name":{"kind":"Name","value":"bildirimYontemi"}},{"kind":"Field","name":{"kind":"Name","value":"kaynakIfade"}}]}},{"kind":"Field","name":{"kind":"Name","value":"cikarimMeta"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"alanYolu"}},{"kind":"Field","name":{"kind":"Name","value":"guven"}},{"kind":"Field","name":{"kind":"Name","value":"kaynakSayfa"}},{"kind":"Field","name":{"kind":"Name","value":"kaynakMadde"}},{"kind":"Field","name":{"kind":"Name","value":"elleDuzeltildi"}}]}},{"kind":"Field","name":{"kind":"Name","value":"bulgular"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"kod"}},{"kind":"Field","name":{"kind":"Name","value":"baslik"}},{"kind":"Field","name":{"kind":"Name","value":"aciklama"}},{"kind":"Field","name":{"kind":"Name","value":"onem"}},{"kind":"Field","name":{"kind":"Name","value":"kaynak"}},{"kind":"Field","name":{"kind":"Name","value":"alanYolu"}}]}},{"kind":"Field","name":{"kind":"Name","value":"denetciSuresi"}}]}}]}}]} as unknown as DocumentNode<SozlesmeAlanGuncelleMutation, SozlesmeAlanGuncelleMutationVariables>;
export const SozlesmeGuncelleDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SozlesmeGuncelle"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"girdi"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SozlesmeGirdi"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sozlesmeGuncelle"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"girdi"},"value":{"kind":"Variable","name":{"kind":"Name","value":"girdi"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"durum"}}]}}]}}]} as unknown as DocumentNode<SozlesmeGuncelleMutation, SozlesmeGuncelleMutationVariables>;
export const UyeDavetEtDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UyeDavetEt"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eposta"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"rol"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Rol"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"uyeDavetEt"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"eposta"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eposta"}}},{"kind":"Argument","name":{"kind":"Name","value":"rol"},"value":{"kind":"Variable","name":{"kind":"Name","value":"rol"}}}]}]}}]} as unknown as DocumentNode<UyeDavetEtMutation, UyeDavetEtMutationVariables>;
export const UyeRolDegistirDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UyeRolDegistir"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"kullaniciId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"rol"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Rol"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"uyeRolDegistir"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"kullaniciId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"kullaniciId"}}},{"kind":"Argument","name":{"kind":"Name","value":"rol"},"value":{"kind":"Variable","name":{"kind":"Name","value":"rol"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"eposta"}},{"kind":"Field","name":{"kind":"Name","value":"rol"}}]}}]}}]} as unknown as DocumentNode<UyeRolDegistirMutation, UyeRolDegistirMutationVariables>;
export const UyeCikarDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UyeCikar"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"kullaniciId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"uyeCikar"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"kullaniciId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"kullaniciId"}}}]}]}}]} as unknown as DocumentNode<UyeCikarMutation, UyeCikarMutationVariables>;
export const AktifPromptDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AktifPrompt"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"tip"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"PromptTipi"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"aktifPrompt"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"tip"},"value":{"kind":"Variable","name":{"kind":"Name","value":"tip"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tip"}},{"kind":"Field","name":{"kind":"Name","value":"icerik"}},{"kind":"Field","name":{"kind":"Name","value":"surum"}},{"kind":"Field","name":{"kind":"Name","value":"aktif"}},{"kind":"Field","name":{"kind":"Name","value":"olusturmaTarihi"}},{"kind":"Field","name":{"kind":"Name","value":"olusturanKullaniciId"}}]}}]}}]} as unknown as DocumentNode<AktifPromptQuery, AktifPromptQueryVariables>;
export const PromptSurumleriDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"PromptSurumleri"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"tip"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"PromptTipi"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"promptSurumleri"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"tip"},"value":{"kind":"Variable","name":{"kind":"Name","value":"tip"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tip"}},{"kind":"Field","name":{"kind":"Name","value":"icerik"}},{"kind":"Field","name":{"kind":"Name","value":"surum"}},{"kind":"Field","name":{"kind":"Name","value":"aktif"}},{"kind":"Field","name":{"kind":"Name","value":"olusturmaTarihi"}},{"kind":"Field","name":{"kind":"Name","value":"olusturanKullaniciId"}}]}}]}}]} as unknown as DocumentNode<PromptSurumleriQuery, PromptSurumleriQueryVariables>;
export const AyarlarDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Ayarlar"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ayarlar"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"denetciRiskEsigi"}},{"kind":"Field","name":{"kind":"Name","value":"maxToken"}},{"kind":"Field","name":{"kind":"Name","value":"guncellemeTarihi"}},{"kind":"Field","name":{"kind":"Name","value":"guncelleyenKullaniciId"}}]}}]}}]} as unknown as DocumentNode<AyarlarQuery, AyarlarQueryVariables>;
export const PromptGuncelleDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"PromptGuncelle"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"tip"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"PromptTipi"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"icerik"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"promptGuncelle"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"tip"},"value":{"kind":"Variable","name":{"kind":"Name","value":"tip"}}},{"kind":"Argument","name":{"kind":"Name","value":"icerik"},"value":{"kind":"Variable","name":{"kind":"Name","value":"icerik"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tip"}},{"kind":"Field","name":{"kind":"Name","value":"icerik"}},{"kind":"Field","name":{"kind":"Name","value":"surum"}},{"kind":"Field","name":{"kind":"Name","value":"aktif"}},{"kind":"Field","name":{"kind":"Name","value":"olusturmaTarihi"}},{"kind":"Field","name":{"kind":"Name","value":"olusturanKullaniciId"}}]}}]}}]} as unknown as DocumentNode<PromptGuncelleMutation, PromptGuncelleMutationVariables>;
export const PromptSurumeDonDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"PromptSurumeDon"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"promptSurumeDon"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tip"}},{"kind":"Field","name":{"kind":"Name","value":"icerik"}},{"kind":"Field","name":{"kind":"Name","value":"surum"}},{"kind":"Field","name":{"kind":"Name","value":"aktif"}},{"kind":"Field","name":{"kind":"Name","value":"olusturmaTarihi"}},{"kind":"Field","name":{"kind":"Name","value":"olusturanKullaniciId"}}]}}]}}]} as unknown as DocumentNode<PromptSurumeDonMutation, PromptSurumeDonMutationVariables>;
export const AyarlariGuncelleDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AyarlariGuncelle"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"denetciRiskEsigi"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Float"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"maxToken"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ayarlariGuncelle"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"denetciRiskEsigi"},"value":{"kind":"Variable","name":{"kind":"Name","value":"denetciRiskEsigi"}}},{"kind":"Argument","name":{"kind":"Name","value":"maxToken"},"value":{"kind":"Variable","name":{"kind":"Name","value":"maxToken"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"denetciRiskEsigi"}},{"kind":"Field","name":{"kind":"Name","value":"maxToken"}},{"kind":"Field","name":{"kind":"Name","value":"guncellemeTarihi"}},{"kind":"Field","name":{"kind":"Name","value":"guncelleyenKullaniciId"}}]}}]}}]} as unknown as DocumentNode<AyarlariGuncelleMutation, AyarlariGuncelleMutationVariables>;