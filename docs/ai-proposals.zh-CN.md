# AI Proposals

本文档说明 `docx update --ai`、本地 AI 命令适配，以及 `.docx.json` 中的 AI 配置。

## 语义更新原则

`docx update --ai` 是 provider-agnostic 的语义 proposal 生成入口。它不会直接修改：

- `.doc/decisions/`
- `.doc/mistakes/`
- module `summary`
- module `riskRules`

AI 只能生成 pending proposal。用户需要通过以下命令显式确认后，语义记忆才会落盘：

```bash
docx proposals accept <id>
```

## 默认 AI Proposal

不提供本地 AI 命令时：

```bash
docx update --staged --ai
```

当前行为：

- 仍然记录 change JSON/Markdown。
- 为受影响模块生成 pending proposal。
- proposal source 为 `ai:provider-agnostic`。
- 不直接修改语义记忆。

## 本地 AI 命令

用户可以让 `docx` 调用本地已有 AI 工具生成 proposal：

```bash
docx update --staged --ai --ai-command "./scripts/docx-local-ai.sh"
docx update --changed --ai --ai-command "ollama run qwen2.5-coder"
docx update --since HEAD~1 --ai --ai-command "claude -p"
```

本地命令通过 stdin 接收稳定 JSON，通过 stdout 返回单个 proposal 或 proposal 数组。

## `.docx.json` 配置

也可以把本地 AI 命令写入 `.docx.json`，之后直接运行 `docx update --ai`：

```json
{
  "ai": {
    "provider": "local-command",
    "command": "codex exec --json",
    "timeoutSeconds": 120,
    "contextSources": ["docx", "codegraph"],
    "output": "proposal-json"
  }
}
```

优先级：命令行 `--ai-command` 高于 `.docx.json.ai.command`。

只有 `ai.provider=local-command` 且 `ai.output` 为空或 `proposal-json` 时，CLI 才会使用配置里的本地命令。

## 输入协议示例

`docx` 会通过 stdin 向本地命令写入：

```json
{
  "schemaVersion": "1.0",
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

本地命令 stdout 可以返回单个 proposal：

```json
{
  "schemaVersion": "1.0",
  "id": "local-ai-chat-summary",
  "type": "module-summary",
  "status": "pending",
  "source": "ai:local-command",
  "evidence": [
    {
      "path": "src/modules/chat/index.ts",
      "reason": "Local AI reviewed the changed chat module."
    }
  ],
  "suggestedTarget": ".doc/modules/chat.json",
  "suggestedPatch": {
    "purpose": "Owns chat conversations."
  }
}
```

也可以返回 proposal 数组。

`docx` 只会写入 schema-valid、`status=pending` 的 proposals；命令失败、输出非 JSON 或 proposal 缺失必要字段时，会返回错误，并且不写入语义记忆。

## 本地工具包装脚本

Codex、Claude、Ollama、Aider 或其他本地命令都可以通过包装脚本接入。包装脚本的职责是：

- 从 stdin 读取 `docx` 输入 JSON。
- 调用本地 AI 工具或 MCP/code graph 工具。
- 将模型输出整理成 `proposal.schema.json` 兼容 JSON。
- 只向 stdout 输出 proposal JSON，不混入解释性文本。

CLI 核心不强制绑定任何 LLM provider。
