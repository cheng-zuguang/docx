import fs from "node:fs";
import path from "node:path";

const input = JSON.parse(fs.readFileSync(0, "utf8"));
const root = input.root;

const report = {
  manifests: [],
  languages: [],
  frameworks: [],
  entrypoints: [],
  imports: [],
  exports: [],
  routes: [],
  testFiles: [],
  configFiles: [],
  moduleCandidates: [],
};

const seen = {
  languages: new Set(),
  frameworks: new Set(),
  imports: new Set(),
  exports: new Set(),
  modules: new Map(),
};

const skipDirs = new Set([".git", ".doc", "dist", "node_modules", "vendor"]);

function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const abs = path.join(dir, entry.name);
    const rel = path.relative(root, abs).split(path.sep).join("/");
    if (entry.isDirectory()) {
      if (!skipDirs.has(entry.name)) walk(abs);
      continue;
    }
    visitFile(abs, rel);
  }
}

function visitFile(abs, rel) {
  const base = path.basename(rel);
  if (base === "package.json") {
    report.manifests.push(rel);
    detectPackageFrameworks(abs);
    return;
  }
  if (!isJSLike(base)) return;

  if (base.endsWith(".ts") || base.endsWith(".tsx")) seen.languages.add("typescript");
  if (base.endsWith(".js") || base.endsWith(".jsx") || base.endsWith(".mjs") || base.endsWith(".cjs")) seen.languages.add("javascript");
  if (isEntrypoint(rel)) report.entrypoints.push(rel);
  if (isTestFile(base)) report.testFiles.push(rel);
  if (isConfigFile(base)) report.configFiles.push(rel);
  if (rel.includes("/routes/") || rel.startsWith("routes/") || rel.startsWith("src/routes/")) report.routes.push(rel);
  const candidate = moduleCandidateForPath(rel);
  if (candidate) seen.modules.set(candidate.name, candidate);

  const source = fs.readFileSync(abs, "utf8");
  detectImports(source);
  detectExports(source);
  detectSourceFrameworks(source);
}

function detectPackageFrameworks(abs) {
  const parsed = JSON.parse(fs.readFileSync(abs, "utf8"));
  const deps = Object.assign({}, parsed.dependencies, parsed.devDependencies, parsed.peerDependencies);
  for (const name of Object.keys(deps)) addFrameworkFromPackage(name);
}

function addFrameworkFromPackage(name) {
  const map = {
    react: "react",
    "react-dom": "react",
    vue: "vue",
    svelte: "svelte",
    next: "next",
    vite: "vite",
    express: "express",
  };
  if (map[name]) seen.frameworks.add(map[name]);
}

function detectSourceFrameworks(source) {
  if (source.includes("from 'react'") || source.includes('from "react"')) seen.frameworks.add("react");
  if (source.includes("from 'vue'") || source.includes('from "vue"')) seen.frameworks.add("vue");
  if (source.includes("from 'express'") || source.includes('from "express"')) seen.frameworks.add("express");
}

function detectImports(source) {
  for (const match of source.matchAll(/\bimport\s+(?:[^'"]+\s+from\s+)?['"]([^'"]+)['"]/g)) seen.imports.add(match[1]);
  for (const match of source.matchAll(/\bexport\s+[^'"]+\s+from\s+['"]([^'"]+)['"]/g)) seen.imports.add(match[1]);
  for (const match of source.matchAll(/\brequire\(['"]([^'"]+)['"]\)/g)) seen.imports.add(match[1]);
}

function detectExports(source) {
  for (const match of source.matchAll(/\bexport\s+(?:default\s+)?(?:async\s+)?(?:function|class|const|let|var|interface|type|enum)\s+([A-Za-z_$][\w$]*)/g)) {
    seen.exports.add(match[1]);
  }
  for (const match of source.matchAll(/\bexport\s*\{([^}]+)\}/g)) {
    for (const item of match[1].split(",")) {
      const name = item.trim().split(/\s+as\s+/)[0].trim();
      if (name) seen.exports.add(name);
    }
  }
  for (const match of source.matchAll(/\bexports\.([A-Za-z_$][\w$]*)\s*=/g)) seen.exports.add(match[1]);
  for (const match of source.matchAll(/\bmodule\.exports\s*=\s*\{([^}]+)\}/g)) {
    for (const item of match[1].split(",")) {
      const name = item.trim().split(":")[0].trim();
      if (name) seen.exports.add(name);
    }
  }
}

function isJSLike(base) {
  return /\.(ts|tsx|js|jsx|mjs|cjs)$/.test(base);
}

function isTestFile(base) {
  return base.includes(".test.") || base.includes(".spec.");
}

function isConfigFile(base) {
  return /\.config\.(ts|js|mjs|cjs)$/.test(base);
}

function isEntrypoint(rel) {
  return ["src/main.ts", "src/main.tsx", "src/index.ts", "src/index.tsx", "src/main.js", "src/index.js"].includes(rel);
}

function moduleCandidateForPath(rel) {
  const parts = rel.split("/");
  for (let i = 0; i + 2 < parts.length; i++) {
    if ((parts[i] === "modules" || parts[i] === "features") && parts[i + 1]) {
      const prefix = parts.slice(0, i + 2).join("/");
      return {
        name: parts[i + 1],
        paths: [`${prefix}/**`],
        confidence: "high",
        reason: `typescript analyzer detected ${parts[i]}/* module convention`,
      };
    }
  }
  if (parts.length >= 2 && parts[0] === "packages" && parts[1]) {
    return {
      name: parts[1],
      paths: [`packages/${parts[1]}/**`],
      confidence: "high",
      reason: "typescript analyzer detected packages/* workspace convention",
    };
  }
  return null;
}

walk(root);

report.languages = Array.from(seen.languages);
report.frameworks = Array.from(seen.frameworks);
report.imports = Array.from(seen.imports);
report.exports = Array.from(seen.exports);
report.moduleCandidates = Array.from(seen.modules.values());

const output = {
  schemaVersion: "1.0",
  analyzer: {
    name: "typescript",
    version: "0.1.0",
    languages: ["typescript", "javascript"],
    capabilities: ["imports", "exports", "frameworks", "routes", "tests", "module-candidates"],
  },
  report,
};

process.stdout.write(JSON.stringify(output));
