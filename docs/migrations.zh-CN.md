# Schema Migration

本文档说明 `docx migrate` 的使用边界。

## 使用场景

当 `.docx.json` 或 `.doc/` 上下文文件的 major schema version 与当前 CLI 不兼容时，普通更新命令不会静默迁移格式，而是停止并提示先运行：

```bash
docx migrate
```

这适合以下场景：

- 升级 `docx` CLI 后发现 schema major version 不兼容。
- 团队成员从旧版本上下文切换到新版本上下文。
- CI 或 `docx doctor` 提示需要显式迁移。

## 当前行为

当前版本会把 `.docx.json` 中的 `schemaVersion` 和 `contextSchemaVersion` 修复到 `1.0`。

迁移保持幂等：重复运行 `docx migrate` 不应改变已经迁移完成的文件。

## 与 `docx update` 的关系

`docx update` 遇到 major schema mismatch 时会停止，并提示运行：

```bash
docx migrate
```

这样可以避免普通代码改动同步流程顺手改写上下文格式，降低多人协作时的意外变更。
