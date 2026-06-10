# Git Hooks

本文档说明 `docx install-hook`、`docx install-agent-hook` 和常见 hook manager 的集成方式。

## 通用 Git Hook

`docx install-hook` 只安装用户显式指定的 hook，不会在 `docx init` 时自动启用。

```bash
docx install-hook pre-commit
docx install-hook post-merge
docx install-hook post-checkout
```

命令会在 `.git/hooks/<hook>` 中插入 docx 托管区块，并保留已有 hook 内容。重复运行会刷新托管区块，不会复制内容。

## 推荐场景

- `pre-commit`：提交前执行 `docx update --staged`，让上下文记录本次 staged changes。
- `post-merge`：合并后执行 `docx update --changed`，记录合并后仍留在工作区的模块改动。
- `post-checkout`：切换分支后执行 `docx update --changed`，记录切换后仍留在工作区的模块改动。

## Agent Lifecycle Hook

如果希望 Codex 或 Claude Code 在 agent 停止响应时自动同步上下文，可以安装 agent hook：

```bash
docx install-agent-hook codex
docx install-agent-hook claude
docx install-agent-hook codex --propose
```

行为：

- `codex`：写入项目级 `.codex/hooks.json`，在 `Stop` 事件运行 `docx finish`。
- `claude`：写入项目级 `.claude/settings.json`，在 `Stop` 事件运行 `docx finish`。
- `--propose`：让 Stop hook 运行 `docx finish --propose`，在记录代码改动后继续创建 proposal task。
- `docx finish` 只在 staged、unstaged 或 untracked 中存在命中 confirmed module 的改动时运行 `docx sync`。
- 已有 hook 事件会被保留；重复安装不会重复追加 `docx finish`。
- `docx doctor` 会把已安装的 Git hook 或 agent lifecycle hook 都计入可选 hook 状态。

## Husky

Husky 项目可在 `.husky/pre-commit` 中显式调用：

```bash
docx update --staged
```

如果希望提交前创建 active-agent proposal task：

```bash
docx update --staged --propose
```

## Lefthook

Lefthook 项目可在 `lefthook.yml` 中配置：

```yaml
pre-commit:
  commands:
    docx:
      run: docx update --staged
```

## 设计边界

Hook 是可选增强，不是默认团队工作流。`docx init` 默认不安装 hook，避免初始化项目时改变用户或团队的提交行为。

`install-agent-hook` 也只安装用户显式选择的 host，不会在 `docx init` 时自动启用。
