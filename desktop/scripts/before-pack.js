"use strict";

const fs = require("node:fs");
const path = require("node:path");

// electron-builder Arch: 0 ia32, 1 x64, 2 armv7l, 3 arm64
function goArch(arch) {
  if (arch === 3 || arch === "arm64") {
    return "arm64";
  }
  return "amd64";
}

module.exports = async function beforePack(context) {
  const repo = path.join(__dirname, "..", "..");
  const desktop = path.join(__dirname, "..");
  const webOut = path.join(repo, "web", "out");
  const destWeb = path.join(desktop, "resources", "web");
  const destBin = path.join(desktop, "resources", "bin");

  if (!fs.existsSync(webOut)) {
    throw new Error("web/out yok; önce web dizininde npm run build çalıştırın");
  }
  fs.rmSync(destWeb, { recursive: true, force: true });
  fs.mkdirSync(destWeb, { recursive: true });
  fs.cpSync(webOut, destWeb, { recursive: true });

  const platform = context.electronPlatformName;
  let srcName;
  let destName;
  if (platform === "darwin") {
    srcName = `api-darwin-${goArch(context.arch)}`;
    destName = "api";
  } else if (platform === "win32") {
    srcName = "api-windows-amd64.exe";
    destName = "api.exe";
  } else {
    throw new Error(`desteklenmeyen platform: ${platform}`);
  }

  const src = path.join(repo, "backend", "bin", srcName);
  if (!fs.existsSync(src)) {
    throw new Error(
      `${srcName} yok; backend dizininde make build-darwin veya make build-windows çalıştırın`,
    );
  }
  fs.rmSync(destBin, { recursive: true, force: true });
  fs.mkdirSync(destBin, { recursive: true });
  const dest = path.join(destBin, destName);
  fs.copyFileSync(src, dest);
  if (platform !== "win32") {
    fs.chmodSync(dest, 0o755);
  }
};
