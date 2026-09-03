"use strict";

const fs = require("node:fs");
const path = require("node:path");

const src = path.join(__dirname, "..", "src", "setup.html");
const destDir = path.join(__dirname, "..", "out");
const dest = path.join(destDir, "setup.html");
fs.mkdirSync(destDir, { recursive: true });
fs.copyFileSync(src, dest);
