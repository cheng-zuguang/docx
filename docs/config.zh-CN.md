# `.docx.json` 配置

`.docx.json` 是 `docx` CLI 在项目根目录生成的工具配置文件。它描述上下文目录、schema version、AI 入口文件，以及可选的本地 AI 命令配置。

`docx init` 默认生成：

```json
{
  "schemaVersion": "1.0",
  "contextDir": ".doc",
  "contextSchemaVersion": "1.0",
  "entryFiles": ["AGENTS.md"],
  "ai": {
    "provider": "",
    "command": "",
    "timeoutSeconds": 120,
    "contextSources": ["docx"],
    "output": "proposal-json"
  }
}
```

## 顶层字段

| 字段 | 类型 | 默认值 | 含义 |
| --- | --- | --- | --- |
| `schemaVersion` | string | `"1.0"` | `.docx.json` 自身的配置 schema version。major version 不兼容时，普通 update 不会静默迁移。 |
| `contextDir` | string | `".doc"` | 项目上下文目录。`docx` 会从这里读取和写入 index、project、modules、changes、proposals 等文件。 |
| `contextSchemaVersion` | string | `"1.0"` | `.doc/` 上下文文件的 schema version。用于判断是否需要 `docx migrate`。 |
| `entryFiles` | string[] | `["AGENTS.md"]` | AI 入口文件列表。`docx init` 会在这些文件中维护 docx 托管区块。 |
| `ai` | object | 见下文 | 可选 AI proposal 配置。当前主要供 `docx update --ai` 使用。 |

## `entryFiles`

`entryFiles` 控制哪些文件会收到 docx 托管区块。

常见配置：

```json
{
  "entryFiles": ["AGENTS.md"]
}
```

```json
{
  "entryFiles": ["AGENTS.md", "CLAUDE.md"]
}
```

初始化行为：

- 显式传 `docx init --entry <file>` 时，使用命令行指定的入口文件。
- 未显式传 `--entry` 时，如果 `AGENTS.md` 或 `CLAUDE.md` 已存在，则只补充已有入口文件。
- 两者都不存在时，只创建 `AGENTS.md`。

## `ai` 字段

`ai` 用于配置本地 AI 命令，让用户不必每次运行 `docx update --ai` 都传 `--ai-command`。

| 字段 | 类型 | 默认值 | 含义 |
| --- | --- | --- | --- |
| `ai.provider` | string | `""` | AI provider 类型。当前只有 `"local-command"` 会触发配置里的本地命令。空字符串表示不启用配置命令。 |
| `ai.command` | string | `""` | 本地命令。`docx` 会把 AI input JSON 写入该命令 stdin，并读取 stdout 中的 proposal JSON。 |
| `ai.timeoutSeconds` | number | `120` | 本地 AI 命令超时时间，单位为秒。 |
| `ai.contextSources` | string[] | `["docx"]` | AI input 的上下文来源标记。当前支持记录 `"docx"`，也可以配置 `"codegraph"` 等外部上下文来源供包装脚本使用。 |
| `ai.output` | string | `"proposal-json"` | 本地命令 stdout 的输出格式。当前支持空字符串或 `"proposal-json"`。 |

## 本地 AI 命令配置示例

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

之后可以直接运行：

```bash
docx update --changed --ai
```

等价于每次手动传入：

```bash
docx update --changed --ai --ai-command "codex exec --json"
```

命令行优先级更高：如果同时存在 `.docx.json.ai.command` 和 `--ai-command`，CLI 使用 `--ai-command`。

## 设计边界

- `.docx.json` 是工具配置，不存项目语义记忆。
- decisions、mistakes、module summary、riskRules 不应通过 `.docx.json` 配置。
- AI 命令只生成 proposals，不直接修改 `.doc/decisions/`、`.doc/mistakes/` 或 module `riskRules`。
- schema major version 不兼容时，应运行 `docx migrate`，不应手动猜测修改版本号。
