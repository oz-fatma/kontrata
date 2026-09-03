import type {
  OrganizasyonumQuery,
  SozlesmeQuery,
  UyelerQuery,
} from "@/generated/graphql";

export type Member = UyelerQuery["uyeler"][number];
export type Organization = NonNullable<OrganizasyonumQuery["organizasyonum"]>;
export type Contract = NonNullable<SozlesmeQuery["sozlesme"]>;
export type ExtractionMeta = NonNullable<NonNullable<Contract["cikarimMeta"]>[number]>;
