import type { CodegenConfig } from "@graphql-codegen/cli";

const config: CodegenConfig = {
  schema: "../backend/graph/schema.graphqls",
  documents: ["graphql/**/*.graphql"],
  generates: {
    "generated/": {
      preset: "client",
      presetConfig: {
        fragmentMasking: false,
      },
      config: {
        scalars: { Time: "string", Upload: "File", JSON: "unknown" },
        skipTypename: true,
        documentMode: "documentNode",
      },
    },
  },
};

export default config;
