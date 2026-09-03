"use strict";

const { spawn } = require("node:child_process");
const path = require("node:path");
const electron = require("electron");

process.env.NODE_ENV = "development";
const child = spawn(electron, ["."], {
  cwd: path.join(__dirname, ".."),
  env: process.env,
  stdio: "inherit",
});
child.on("exit", (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code ?? 0);
});
