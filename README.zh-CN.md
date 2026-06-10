# docx

[English](README.md) | 中文

`docx` 是一个 Go CLI，用来为任意项目创建和持续维护 AI 可读的项目上下文。

它会在仓库中生成 `.doc/` 上下文目录和 `.docx.json` 配置文件。AI agent 先读取很小的
`.doc/index.json`，再按 `readOrder`、`moduleMap` 渐进读取当前任务真正需要的项目、模块、决策、错题、变更或 proposal。

## 适合什么场景

当一个仓库已经有足够多的结构、历史和隐性约定，导致每次新的 AI 会话都要重新摸索上下文时，可以使用 `docx`。

常见场景：

- **新 AI 会话快速上手**：初始化 `.doc/` 后，agent 不需要一开始扫描整个仓库。
- **团队共享项目记忆**：把项目事实、模块边界、决策、错题和近期变更沉淀为可提交的文件。
- **提交前同步上下文**：基于 staged changes 写入变更记录，让项目上下文跟代码保持接近。
- **大型仓库或模块化仓库**：通过 `moduleMap` 把文件路径路由到对应模块上下文，减少无关阅读。
- **AI 辅助语义更新**：当前 agent 可以通过 task 文件起草语义 proposal，但长期记忆必须显式确认。
- **多语言项目**：先用 generic scanner 获得基础扫描能力，再逐步接入语言 analyzer。

## 核心原则

`docx` 区分自动生成事实和已确认语义记忆。

- generated facts 可以由 CLI 自动刷新。
- decisions、mistakes、module summary、riskRules 等语义记忆需要用户确认或 accept proposal 后才落盘。
- `AGENTS.md`、`CLAUDE.md`、`.gitignore` 和 git hook 都使用托管区块，避免覆盖用户已有内容。

## 安装

macOS 和 Linux：

```bash
curl -fsSL https://raw.githubusercontent.com/cheng-zuguang/docx/main/install.sh | sh
```

Windows PowerShell：

```powershell
iwr https://raw.githubusercontent.com/cheng-zuguang/docx/main/install.ps1 -UseB | iex
```

npm：

```bash
npm install -g @chengzg/docx
docx --help
```

npx：

```bash
npx @chengzg/docx --help
npx @chengzg/docx init
```

开发期运行：

```bash
go test ./...
go run ./cmd/docx --help
go run ./cmd/docx init
```

## 生成的结构

默认结构和文件含义：

```text
.docx.json                    # CLI 配置：上下文目录、schema version、入口文件
.doc/
|-- index.json                # 上下文路由器：项目摘要、readOrder、moduleMap
|-- project.json              # 项目级生成事实和高层摘要
|-- capabilities.json         # 当前可用 analyzer 和推荐但缺失的 analyzer
|-- README.md                 # 给人的本地说明：该目录由 docx 生成
|-- rules/
|   `-- agent.md              # AI agent 的上下文读取协议
|-- modules/
|   `-- <module>.json         # 模块上下文：路径、状态、事实、读取提示、风险规则、近期变更
|-- changes/
|   |-- index.json            # 可重建的变更记录索引
|   |-- <change-id>.json      # 机器可读变更记录
|   `-- <change-id>.md        # 人类可读变更记录
|-- proposals/
|   |-- index.json            # 可重建的语义更新 proposal 索引
|   |-- <proposal-id>.json    # 机器可读 pending/accepted/rejected proposal
|   `-- <proposal-id>.md      # 人类可读 proposal 审计说明
|-- decisions/
|   `-- index.json            # 可重建的已确认设计决策索引
|-- mistakes/
|   `-- index.json            # 可重建的已确认反模式经验索引
|-- .cache/                   # 本地缓存，git 忽略
|-- local/                    # 本机/用户状态，git 忽略
`-- tmp/                      # 临时文件，git 忽略
```

`.doc/.cache/`、`.doc/local/`、`.doc/tmp/` 是本地状态，会通过托管 `.gitignore` 区块忽略。

## 命令说明

### `docx init`

初始化项目上下文。

```bash
docx init
docx init --dir .context
docx init --entry AGENTS.md --entry CLAUDE.md
docx init --non-interactive
docx init --accept-candidates
docx init --interactive
docx init --accept-candidates --summarize
docx apply init .doc/tmp/init-summary-output.json
```

行为：

- 创建 `.docx.json` 和 `.doc/` 目录结构。
- 扫描 manifest、语言、框架、入口、测试、配置文件和模块候选。
- 向 AI 入口文件插入托管区块。
- 未显式传 `--entry` 时，优先补充已有 `AGENTS.md` / `CLAUDE.md`；两者都不存在时只创建 `AGENTS.md`。
- 默认不安装 git hook。

Agent 辅助初始化：

- `--accept-candidates` 会跳过交互确认，直接接受 scanner 发现的模块候选。
- `--summarize` 会写入 task 文件：`.doc/tmp/init-summary-input.json` / `.doc/tmp/init-summary-prompt.md`，交给当前 agent 生成摘要。
- `docx` 永远不会主动调用 AI 命令；当前 agent 写出结果后，再由 `docx apply init` 应用。
- 应用 AI 输出时只写 `project.summary` 和 `module.summary`；不会自动写 `riskRules`、`decisions` 或 `mistakes`。

AI 输出文件格式：

```json
{
  "schemaVersion": "1.0",
  "project": {
    "summary": "AI-readable project summary."
  },
  "modules": [
    {
      "name": "chat",
      "summary": {
        "purpose": "Owns chat workflows.",
        "ownedConcepts": ["messages", "threads"],
        "nonGoals": []
      }
    }
  ]
}
```

也可以直接通过 stdin 应用：

```bash
docx apply init --stdin < .doc/tmp/init-summary-output.json
```

交互式模块确认示例：

```text
accept chat
ignore temp
rename billing payments
merge platform api,web
done
```

当 scanner 识别出的模块边界需要人工确认时，使用 `--interactive`。后续 `docx update` 会依赖这些边界把改动映射到模块上下文。

### `docx scan`

查看 `docx` 能识别到什么，不修改语义记忆。

```bash
docx scan
docx scan --json
docx scan --analyzer generic
docx scan --analyzer typescript
docx scan --analyzer ./tools/my-analyzer
```

适合用来调试项目发现结果、验证 analyzer 输出，或在 `init --interactive` 前查看候选模块。

### `docx update`

基于代码改动记录 change，并刷新生成事实。

```bash
docx update --staged
docx update --changed
docx update --since HEAD~1
docx update --module chat
```

`--changed` 会记录所有未提交改动：staged、unstaged 和 untracked 文件中命中 confirmed module 的部分。

典型工作流：

```bash
# 提交前
docx update --staged

# 本地修改某个模块后
docx update --changed

# 合并分支后
docx update --since main
```

该命令会在 `.doc/changes/` 下写入 JSON 和 Markdown 两种 change record，并更新受影响模块的 `recentChanges`。

### `docx sync`

同步当前 agent 完成后的工作区上下文。

```bash
docx sync
```

`sync` 会记录 staged、unstaged 和 untracked 中命中 confirmed module 的改动，刷新 deterministic module facts，并写入 `.doc/tmp/agent-sync.md`，提醒当前 agent 继续处理需要语义判断的 follow-up。

### `docx finish`

为 Codex / Claude Code 这类 agent lifecycle hook 准备的安全收尾命令。

```bash
docx finish
```

`finish` 会先检查 staged、unstaged 和 untracked 中是否有命中 confirmed module 的改动。有改动时运行 `docx sync`；没有改动或只有未映射文件时直接成功退出，避免 Stop hook 在空会话里重复写旧 change。

### `docx update --propose`

创建语义 proposal task，但不直接修改已确认记忆。

```bash
docx update --staged --propose
docx update --changed --propose
docx update --since HEAD~1 --propose
```

使用 `--propose` 时，`docx` 会创建 task 文件（`.doc/tmp/proposals-input.json` 和 `.doc/tmp/proposals-prompt.md`），交给当前 agent 生成 proposal：

```bash
docx apply proposals .doc/tmp/proposals-output.json
```

`docx init` 写入的 agent 协议会提醒 AI 工具：如果已经安装 lifecycle hook，就让它运行 `docx finish`；否则在结束前手动运行 `docx sync`。当希望当前 agent 顺手准备语义 proposal 时，使用 `docx update --changed --propose` 生成 task。

### `docx proposals`

查看、接受或拒绝语义 proposal。

```bash
docx proposals list
docx proposals show <id>
docx proposals accept <id>
docx proposals accept <id> --target .doc/mistakes/testing.md
docx proposals reject <id>
```

这个流程用于保证长期项目记忆是有意识沉淀的。accepted proposal 可以更新 decisions、mistakes、module summaries 或 module riskRules；rejected proposal 会保留审计历史。

### `docx index`

重建或检查索引。

```bash
docx index
docx index --check
docx index --section changes
docx index --section proposals
docx index --section decisions
docx index --section mistakes
```

适合在 merge 后、手动编辑 `.doc/` 后、或 proposal 状态变化后使用。

### `docx doctor`

检查当前 `docx` 上下文是否健康。

```bash
docx doctor
docx doctor --json
docx doctor --strict
```

它会检查配置、必要文件、schema version、索引、analyzer capabilities，以及可选 Git hook 或 agent hook 状态。`--strict` 适合放到 CI。

### `docx migrate`

显式执行 schema migration。

```bash
docx migrate
```

当 major schema mismatch 出现时，`docx update` 会停止并提示运行 migration，不会静默改变上下文格式。

### `docx install-hook`

安装可选 git hook，并使用托管区块保留已有内容。

```bash
docx install-hook pre-commit
docx install-hook pre-commit --propose
docx install-hook post-merge
docx install-hook post-checkout
```

`pre-commit` 默认写入 `docx update --staged`。使用 `--propose` 时写入 `docx update --staged --propose`，让提交前创建 active-agent proposal task 文件。`post-merge` 和 `post-checkout` 会写入 `docx update --changed`，用于在分支移动后记录未提交的模块改动。重复运行会刷新托管区块，不会重复追加。

Hook manager 示例：

```bash
# Husky pre-commit 内容
docx update --staged
# 或启用 agent proposal
docx update --staged --propose
```

```yaml
# lefthook.yml
pre-commit:
  commands:
    docx:
      run: docx update --staged
```

### `docx install-agent-hook`

安装可选的 agent lifecycle hook，让 Codex 或 Claude Code 在 Stop 事件自动运行 `docx finish`。

```bash
docx install-agent-hook codex
docx install-agent-hook claude
docx install-agent-hook codex --propose
```

`codex` 会写入项目级 `.codex/hooks.json`；`claude` 会写入项目级 `.claude/settings.json`。已有 hook 事件会被保留，重复运行不会重复追加 `docx finish`。使用 `--propose` 时，Stop hook 会运行 `docx finish --propose`，在记录代码改动后继续创建 active-agent proposal task。

## 端到端使用案例

### 从当前源码本地打包安装

仅用于本地调试，不依赖 GitHub release：

```bash
npm run install:local
docx --help
```

如果不想写入系统级 npm 全局目录，可以安装到临时 prefix：

```bash
DOCX_LOCAL_PREFIX=/tmp/docx-local npm run install:local
/tmp/docx-local/bin/docx --help
```

该脚本会从当前工作区执行 `go build ./cmd/docx`，把二进制放入 npm 包的 `npm/bin-runtime/`，再通过 `npm pack` 和本地 `.tgz` 安装。安装过程中会设置 `DOCX_SKIP_DOWNLOAD=1`，因此不会下载 release 产物。

### 接入已有项目

在已有仓库根目录运行：

```bash
# 1. 让 docx 扫描项目、接受检测到的模块候选，并写入当前 agent 初始化 task。
docx init --accept-candidates --summarize

# 2. 让当前 agent 根据 task 生成输出，然后应用：
#    docx apply init .doc/tmp/init-summary-output.json

# 3. 检查上下文健康状态，并提交上下文文件。
docx doctor
git add .doc .docx.json AGENTS.md .gitignore
git commit -m "Initialize docx project context"
```

之后正常修改代码，并在提交前同步上下文：

```bash
git add path/to/changed/files
docx update --staged
git add .doc/changes .doc/modules
git commit -m "Implement feature"
```

如果后续希望 AI 辅助沉淀长期记忆，走 proposal 流程：

```bash
docx update --changed --propose
# 当前 agent 完成 task 后执行：
# docx apply proposals .doc/tmp/proposals-output.json
docx proposals list
docx proposals show <id>
docx proposals accept <id>
```

初始化阶段只会应用项目摘要和模块摘要。decisions、mistakes、riskRules 属于长期判断类记忆，应通过 proposal 或用户确认后再落库。

### 为仓库初始化 AI 上下文

```bash
docx init --interactive
git add .doc .docx.json AGENTS.md .gitignore
```

之后 AI agent 可以先读 `AGENTS.md`，再根据托管区块进入 `.doc/index.json`，按需渐进读取上下文。

如果希望初始化时让当前 agent 生成有用摘要：

```bash
docx init --accept-candidates --summarize
# 当前 agent 完成 task 后执行：
# docx apply init .doc/tmp/init-summary-output.json
git add .doc .docx.json AGENTS.md .gitignore
```

这会自动确认模块候选，并为项目摘要和模块摘要写入 task 文件。长期判断类记忆仍然保持受控：`riskRules`、`decisions`、`mistakes` 不会在初始化时自动落库。

### 每次提交前同步项目上下文

```bash
git add internal/cli/update.go internal/cli/update_test.go
docx update --staged
git add .doc/changes .doc/modules
```

这样可以记录本次改动，并把 change 引用挂到受影响模块上。

### 使用当前 AI agent 生成语义更新建议

```bash
docx update --changed --propose
# 当前 agent 完成 task 后执行：
# docx apply proposals .doc/tmp/proposals-output.json
docx proposals list
docx proposals show <id>
docx proposals accept <id>
```

当前 AI agent 负责判断语义更新，`docx` 负责校验、落盘 proposal 和显式确认。

### merge 后恢复索引健康

```bash
docx index
docx doctor
```

当 teammate 修改了 `.doc/`，或 merge 后 index 可能过期时，可以用这组命令恢复健康状态。

## Analyzer 支持

当前 analyzer 层级：

- `generic`：Go CLI 内置，适用于任意仓库。
- `typescript` / `javascript`：可选 Node analyzer，识别 package manifest、imports、exports、framework signals、可行范围内的 routes 和 tests。
- 外部 analyzer：通过 analyzer plugin protocol 调用。

路线图见 [docs/analyzers/protocol.zh-CN.md](docs/analyzers/protocol.zh-CN.md)。

## 更多文档

- [English README](README.md)
- [中文 PRD](.scratch/docx-generalization/PRD.zh-CN.md)
- [配置项说明](docs/config.zh-CN.md)
- [Schema 文档](docs/schema/README.zh-CN.md)
- [Schema Migration](docs/migrations.zh-CN.md)
- [Git Hooks](docs/hooks.zh-CN.md)
- [Agent Proposals](docs/ai-proposals.zh-CN.md)
- [发布流程](docs/release.zh-CN.md)
- [Analyzer 协议](docs/analyzers/protocol.zh-CN.md)
