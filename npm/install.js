#!/usr/bin/env node

const fs = require("fs");
const https = require("https");
const os = require("os");
const path = require("path");
const childProcess = require("child_process");
const zlib = require("zlib");

const repo = process.env.DOCX_REPO || "cheng-zuguang/docx";
const version = process.env.DOCX_VERSION || "latest";
const runtimeDir = path.join(__dirname, "bin-runtime");

if (process.env.DOCX_SKIP_DOWNLOAD) {
  fs.mkdirSync(runtimeDir, { recursive: true });
  console.log("DOCX_SKIP_DOWNLOAD is set; skipping docx binary download.");
  process.exit(0);
}

const platformMap = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};
const archMap = {
  x64: "amd64",
  arm64: "arm64",
};

const platform = platformMap[process.platform];
const arch = archMap[process.arch];
if (!platform || !arch) {
  console.error(`Unsupported platform: ${process.platform}/${process.arch}`);
  process.exit(1);
}

const ext = platform === "windows" ? "zip" : "tar.gz";
const asset = `docx_${platform}_${arch}.${ext}`;
const url =
  version === "latest"
    ? `https://github.com/${repo}/releases/latest/download/${asset}`
    : `https://github.com/${repo}/releases/download/${version}/${asset}`;

fs.mkdirSync(runtimeDir, { recursive: true });

const archive = path.join(os.tmpdir(), `${Date.now()}-${asset}`);
download(url, archive)
  .then(() => extract(archive, runtimeDir, platform))
  .then(() => {
    fs.rmSync(archive, { force: true });
    console.log(`Installed docx ${platform}/${arch} binary.`);
  })
  .catch((error) => {
    fs.rmSync(archive, { force: true });
    console.error(error.message || error);
    process.exit(1);
  });

function download(url, destination) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(destination);
    https
      .get(url, (response) => {
        if (
          response.statusCode >= 300 &&
          response.statusCode < 400 &&
          response.headers.location
        ) {
          file.close();
          fs.rmSync(destination, { force: true });
          download(response.headers.location, destination).then(resolve, reject);
          return;
        }
        if (response.statusCode !== 200) {
          file.close();
          reject(new Error(`Failed to download ${url}: HTTP ${response.statusCode}`));
          return;
        }
        response.pipe(file);
        file.on("finish", () => file.close(resolve));
      })
      .on("error", (error) => {
        file.close();
        reject(error);
      });
  });
}

function extract(archive, destination, platform) {
  if (platform === "windows") {
    childProcess.execFileSync("powershell", [
      "-NoProfile",
      "-ExecutionPolicy",
      "Bypass",
      "-Command",
      `Expand-Archive -Path '${archive}' -DestinationPath '${destination}' -Force`,
    ]);
    return;
  }

  const tarBytes = zlib.gunzipSync(fs.readFileSync(archive));
  extractTarSingleBinary(tarBytes, destination);
}

function extractTarSingleBinary(buffer, destination) {
  for (let offset = 0; offset + 512 <= buffer.length; offset += 512) {
    const header = buffer.subarray(offset, offset + 512);
    const name = header.toString("utf8", 0, 100).replace(/\0.*$/, "");
    if (!name) {
      break;
    }
    const sizeText = header.toString("utf8", 124, 136).replace(/\0.*$/, "").trim();
    const size = parseInt(sizeText || "0", 8);
    const dataStart = offset + 512;
    const dataEnd = dataStart + size;
    if (path.basename(name) === "docx") {
      const target = path.join(destination, "docx");
      fs.writeFileSync(target, buffer.subarray(dataStart, dataEnd), { mode: 0o755 });
      return;
    }
    offset = dataStart + Math.ceil(size / 512) * 512 - 512;
  }
  throw new Error("Archive did not contain docx binary.");
}
