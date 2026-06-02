# TypeScript Node Analyzer

`docx` 内置一个可选的 Node analyzer，用于 TypeScript 和 JavaScript 项目。Go CLI 仍然是稳定核心；当本机有可用 `node` 时，`docx scan --analyzer typescript` 会通过 analyzer protocol 启动 Node analyzer。

## 使用

```bash
docx scan --analyzer typescript
docx scan --analyzer typescript --json
docx scan --analyzer javascript --json
```

`typescript` 和 `javascript` 当前指向同一个 bundled analyzer。

## 能力

当前 analyzer 识别：

- `package.json`
- TypeScript 和 JavaScript 语言信号
- React、Vue、Svelte、Next、Vite、Express framework signals
- `src/main.*`、`src/index.*` entrypoints
- ESM imports、re-exports、CommonJS `require`
- ESM exports、CommonJS `exports.foo` 和 `module.exports = { foo }`
- `src/routes/**`、`routes/**` 等可行范围内的 route files
- `.test.*`、`.spec.*` test files
- `.config.*` config files
- `modules/*`、`features/*`、`packages/*` module candidates

## 启用和降级

- Node 可用时，包含 JS/TS 文件的项目会在 `.doc/capabilities.json` 中记录 `typescript` analyzer 为 available。
- Node 不可用时，`typescript` analyzer 不会启用；CLI 会回退到 generic scanning。
- analyzer 失败或输出协议不兼容时，CLI 会输出 `Analyzer diagnostic:`，提示安装或配置 analyzer 后重试。

## 边界

当前版本使用轻量文本扫描，不依赖 TypeScript compiler API。它适合作为第一阶段 analyzer protocol 的真实 Node 集成；更深的 AST、tsconfig、path alias、framework route graph 和 symbol-level import/export 分析留给后续增强。
