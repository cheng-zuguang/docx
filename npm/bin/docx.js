#!/usr/bin/env node

const childProcess = require("child_process");
const path = require("path");

const exe = process.platform === "win32" ? "docx.exe" : "docx";
const binary = path.join(__dirname, "..", "bin-runtime", exe);

const result = childProcess.spawnSync(binary, process.argv.slice(2), {
  stdio: "inherit",
});

if (result.error) {
  console.error(`docx binary was not found at ${binary}`);
  console.error("Run `npm rebuild @cheng-zuguang/docx` or reinstall the package.");
  process.exit(1);
}

if (typeof result.status === "number") {
  process.exit(result.status);
}
process.exit(1);
