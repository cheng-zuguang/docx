# Agent Proposals

本文档说明 `docx update --propose`、active-agent task，以及语义 proposal 的输入/输出协议。

## 语义更新原则

`docx update --propose` 是语义 proposal 的 task 入口。它不会直接修改：

- `.doc/decisions/`
- `.doc/mistakes/`
- module `summary`
- module `riskRules`
- `.doc/index.json` 的 confirmed `moduleMap`

`docx` 不主动调用 AI 命令。当前 agent 读取 `.doc/tmp/*-prompt.md` 和 `.doc/tmp/*-input.json`，生成 JSON 输出，再由 `docx apply proposals` 写入 pending proposal。用户需要通过以下命令显式确认后，语义记忆才会落盘：

```bash
docx proposals accept <id>
```

## Proposal Task

```bash
docx update --staged --propose
docx update --changed --propose
docx update --since HEAD~1 --propose
```

当前行为：

- 记录 change JSON/Markdown。
- 创建 `.doc/tmp/proposals-input.json`。
- 创建 `.doc/tmp/proposals-prompt.md`。
- 不直接写 proposal，不直接修改语义记忆。

当前 agent 完成 proposal JSON 后应用：

```bash
docx apply proposals .doc/tmp/proposals-output.json
```

也可以从 stdin 应用：

```bash
docx apply proposals --stdin < .doc/tmp/proposals-output.json
```

## 输入协议示例

`docx` 会把稳定 JSON 写入 `.doc/tmp/proposals-input.json`：

```json
{
  "schemaVersion": "1.0",
  "changeId": "20260101T000000.000000000Z",
  "source": "git:staged",
  "modules": [
    {
      "name": "chat",
      "status": "confirmed",
      "paths": ["src/modules/chat/**"],
      "summary": {
        "purpose": "...",
        "ownedConcepts": [],
        "nonGoals": []
      },
      "readHints": {
        "alwaysRead": [],
        "readFor": []
      },
      "riskRules": [],
      "recentChanges": []
    }
  ],
  "files": [
    {
      "path": "src/modules/chat/index.ts",
      "changeType": "modified",
      "signals": ["sourceTouched"]
    }
  ]
}
```

## 输出协议示例

当前 agent 可以返回一个 proposal：

```json
{
  "schemaVersion": "1.0",
  "changeId": "20260101T000000.000000000Z",
  "proposals": {
    "schemaVersion": "1.0",
    "id": "active-agent-chat-summary",
    "type": "module-summary",
    "status": "pending",
    "source": "ai:active-agent",
    "evidence": [
      {
        "path": "src/modules/chat/index.ts",
        "reason": "Active agent reviewed the changed chat module."
      }
    ],
    "suggestedTarget": ".doc/modules/chat.json",
    "suggestedPatch": {
      "purpose": "Owns chat conversations."
    }
  }
}
```

也可以返回 proposal 数组。

`docx` 只会写入 schema-valid、`status=pending`、带 evidence 的 proposals；输出非 JSON 或 proposal 缺失必要字段时，会返回错误，并且不写入语义记忆。

## 模块分区 Proposal

当 scanner 发现的模块粒度太粗时，当前 agent 应生成 `module-partition` proposal，而不是直接改 `.doc/index.json`：

```json
{
  "schemaVersion": "1.0",
  "id": "partition-cli-context",
  "type": "module-partition",
  "status": "pending",
  "source": "ai:active-agent",
  "evidence": [
    {
      "path": "internal/cli/sync.go",
      "reason": "Context synchronization is a separate workflow from command dispatch."
    }
  ],
  "suggestedTarget": ".doc/index.json",
  "suggestedPatch": {
    "modules": [
      {
        "name": "context-sync",
        "paths": ["internal/cli/sync.go", "internal/cli/update.go"],
        "purpose": "Owns change records and active-agent context synchronization.",
        "ownedConcepts": ["change record", "recentChanges", "agent sync task"],
        "nonGoals": ["command dispatch"]
      }
    ]
  }
}
```

接受该 proposal 后，`docx` 会更新 `.doc/index.json` 的 `moduleMap`，并写入对应 `.doc/modules/<module>.json`。
