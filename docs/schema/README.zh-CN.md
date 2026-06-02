# docx Context Schema v1

本文档定义 `docx` 第一版上下文契约。目标是让 CLI、AI agent 和人类维护者对 `.doc/` 中每类文件的职责、可写边界和 schema 版本有一致理解。

## Schema Version

第一版上下文 schema version 固定为：

```text
1.0
```

所有机器可读 JSON 文件都必须包含：

```json
{
  "schemaVersion": "1.0"
}
```

重大不兼容升级必须通过显式 `docx migrate` 完成，普通 `docx update` 不得静默执行 major migration。

## Required Directory Structure

默认上下文目录为 `.doc/`，可通过 `.docx.json` 的 `contextDir` 配置为其他目录。

```text
.doc/
  index.json
  project.json
  capabilities.json
  README.md
  rules/
    agent.md
  modules/
    <module>.json
  changes/
    index.json
    <change-id>.json
    <change-id>.md
  proposals/
    index.json
    <proposal-id>.json
    <proposal-id>.md
  decisions/
    index.json
    ADR-0001-<slug>.md
  mistakes/
    index.json
    runtime-environment.md
    api-contracts.md
    data-migration.md
    concurrency.md
    security.md
    testing.md
    project-specific.md
  .cache/
  local/
  tmp/
```

## Automation Boundaries

`docx` 将上下文分成两类：自动事实和语义记忆。

### 可由自动化覆盖

这些文件属于 facts 或可再生成索引，CLI 可以在对应命令中重写：

- `.docx.json`
- `.doc/index.json`
- `.doc/project.json`
- `.doc/capabilities.json`
- `.doc/changes/index.json`
- `.doc/proposals/index.json`
- `.doc/decisions/index.json`
- `.doc/mistakes/index.json`
- `.doc/modules/<module>.json` 中的 `facts` 和 `recentChanges`

### 只能确认后修改

这些内容属于长期语义记忆，不能被扫描或 AI 静默覆盖：

- `.doc/decisions/*.md`
- `.doc/mistakes/*.md`
- `.doc/modules/<module>.json` 中的 `summary`
- `.doc/modules/<module>.json` 中的 `riskRules`
- `.doc/modules/<module>.json` 中人工确认过的 `readHints`

未确认的语义更新必须写入 `.doc/proposals/`，再由用户执行 accept/reject。

## Machine Schemas

机器可校验 schema 放在：

```text
schemas/v1/
  index.schema.json
  project.schema.json
  module.schema.json
  capabilities.schema.json
  change.schema.json
  proposal.schema.json
  decision-index.schema.json
  mistake-index.schema.json
  analyzer-input.schema.json
  analyzer-output.schema.json
```

Analyzer 插件协议详见 `docs/analyzers/protocol.zh-CN.md`。该协议约定 CLI 通过 stdin/stdout 与外部 analyzer 交换 JSON，并在外部 analyzer 失败或协议不兼容时回退到 generic scanning。

## Mistake Categories

默认 mistakes 分类按反模式类型组织，不按 frontend/backend/语言组织：

- `runtime-environment.md`
- `api-contracts.md`
- `data-migration.md`
- `concurrency.md`
- `security.md`
- `testing.md`
- `project-specific.md`

单条 mistake 应通过 `appliesTo` 或正文标签表达适用技术栈，例如 React Native、Node、SSR、CLI、Go、Python，而不是把文件名命名为 `frontend.md`。

## Example index.json

```json
{
  "schemaVersion": "1.0",
  "project": {
    "name": "example",
    "type": ["cli"],
    "primaryLanguages": ["go"],
    "summary": "AI-readable project context managed by docx."
  },
  "readOrder": [
    {
      "when": "always",
      "files": [".doc/project.json", ".doc/rules/agent.md"]
    },
    {
      "when": "editing module",
      "resolve": ".doc/modules/{module}.json"
    }
  ],
  "moduleMap": {
    "cli": {
      "paths": ["cmd/**", "internal/cli/**"],
      "context": ".doc/modules/cli.json",
      "confidence": "confirmed"
    }
  }
}
```

## Example module.json

```json
{
  "schemaVersion": "1.0",
  "module": "cli",
  "status": "confirmed",
  "paths": ["cmd/**", "internal/cli/**"],
  "summary": {
    "purpose": "Command-line interface for initializing and maintaining project context.",
    "ownedConcepts": ["command", "context initialization"],
    "nonGoals": ["language-specific deep analysis"]
  },
  "facts": {
    "entrypoints": ["cmd/docx/main.go"],
    "publicApi": ["docx init"],
    "dependencies": [],
    "dependents": [],
    "tests": ["internal/cli/*_test.go"],
    "lastScannedAt": ""
  },
  "readHints": {
    "alwaysRead": ["internal/cli/cli.go"],
    "readFor": [
      {
        "when": "init behavior",
        "files": ["internal/cli/init_test.go", "internal/cli/cli.go"]
      }
    ]
  },
  "riskRules": [
    "Do not overwrite user-authored content outside managed blocks."
  ],
  "recentChanges": []
}
```
