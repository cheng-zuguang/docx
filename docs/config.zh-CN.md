# `.docx.json` 配置

`.docx.json` 是 `docx` CLI 在项目根目录生成的工具配置文件。它描述上下文目录、schema version 和 AI 入口文件。

`docx init` 默认生成：

```json
{
  "schemaVersion": "1.0",
  "contextDir": ".doc",
  "contextSchemaVersion": "1.0",
  "entryFiles": ["AGENTS.md"]
}
```

## 顶层字段

| 字段 | 类型 | 默认值 | 含义 |
| --- | --- | --- | --- |
| `schemaVersion` | string | `"1.0"` | `.docx.json` 自身的配置 schema version。major version 不兼容时，普通 update 不会静默迁移。 |
| `contextDir` | string | `".doc"` | 项目上下文目录。`docx` 会从这里读取和写入 index、project、modules、changes、proposals 等文件。 |
| `contextSchemaVersion` | string | `"1.0"` | `.doc/` 上下文文件的 schema version。用于判断是否需要 `docx migrate`。 |
| `entryFiles` | string[] | `["AGENTS.md"]` | AI 入口文件列表。`docx init` 会在这些文件中维护 docx 托管区块。 |

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

## 设计边界

- `.docx.json` 是工具配置，不存项目语义记忆。
- decisions、mistakes、module summary、riskRules 不应通过 `.docx.json` 配置。
- `docx` 不通过配置主动调用 AI 命令；语义更新通过 `.doc/tmp/` task 交给当前 agent。
- schema major version 不兼容时，应运行 `docx migrate`，不应手动猜测修改版本号。
