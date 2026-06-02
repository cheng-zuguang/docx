# Analyzer Plugin Protocol v1

本文档定义 `docx` Go CLI 与语言专用 analyzer 的最小 JSON 协议。协议目标是让 CLI 保持跨语言核心能力，同时允许 Node、Go、Python 等生态提供更深的项目理解。

## 调用模型

`docx scan --analyzer <command>` 会启动 `<command>`，通过 stdin 写入 analyzer input JSON，并从 stdout 读取 analyzer output JSON。

内置 analyzer 名称为 `generic`：

```bash
docx scan --analyzer generic
docx scan --analyzer typescript --json
docx scan --analyzer ./tools/docx-ts-analyzer --json
```

如果外部 analyzer 退出失败、输出非 JSON，或 `schemaVersion` 不兼容，CLI 应尽量回退到 generic scanning。

`typescript` 和 `javascript` 是当前内置的 bundled Node analyzer 名称，详见 `docs/analyzers/typescript-node.zh-CN.md`。

## Input JSON

Schema: `schemas/v1/analyzer-input.schema.json`

```json
{
  "schemaVersion": "1.0",
  "root": "/absolute/project/root"
}
```

字段说明：

- `schemaVersion`: 当前固定为 `1.0`。
- `root`: 目标项目根目录的绝对路径。analyzer 应只读取该目录下的项目文件。

## Output JSON

Schema: `schemas/v1/analyzer-output.schema.json`

```json
{
  "schemaVersion": "1.0",
  "analyzer": {
    "name": "typescript",
    "version": "0.1.0",
    "languages": ["typescript", "javascript"],
    "capabilities": ["imports", "exports", "frameworks", "routes", "tests"]
  },
  "report": {
    "manifests": ["package.json"],
    "languages": ["typescript"],
    "frameworks": ["react"],
    "entrypoints": ["src/index.ts"],
    "imports": ["react"],
    "exports": ["App"],
    "routes": ["src/routes/home.ts"],
    "testFiles": ["src/index.test.ts"],
    "configFiles": ["vite.config.ts"],
    "moduleCandidates": [
      {
        "name": "chat",
        "paths": ["src/modules/chat/**"],
        "confidence": "high",
        "reason": "analyzer detected module ownership"
      }
    ]
  }
}
```

## Capabilities JSON

`.doc/capabilities.json` 使用稳定对象格式记录 analyzer 状态：

```json
{
  "schemaVersion": "1.0",
  "availableAnalyzers": [
    {
      "name": "generic",
      "kind": "builtin",
      "languages": ["*"],
      "capabilities": ["manifests", "languages", "frameworks", "entrypoints", "tests", "module-candidates"],
      "status": "available"
    }
  ],
  "missingRecommendedAnalyzers": [
    {
      "name": "typescript",
      "kind": "external",
      "languages": ["typescript", "javascript"],
      "capabilities": ["imports", "exports", "frameworks", "routes", "tests"],
      "status": "missing",
      "installHint": "Install or configure the optional Node analyzer."
    }
  ],
  "lastCheckedAt": ""
}
```

## Roadmap

P0:

- Generic builtin analyzer.
- TypeScript/JavaScript Node analyzer.
- Go analyzer.

P1:

- Python analyzer.
- Rust analyzer.
- Java/Kotlin analyzer.

P2:

- .NET analyzer.
- PHP analyzer.
- Ruby analyzer.

## Conformance

协议一致性测试必须覆盖：

- CLI 向 analyzer stdin 写入 `schemaVersion` 和绝对 `root`。
- CLI 能消费合法 analyzer output。
- analyzer 失败时回退 generic scanning。
- analyzer 输出不兼容 schemaVersion 时回退 generic scanning。
