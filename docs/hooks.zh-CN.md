# Git Hooks

本文档说明 `docx install-hook` 和常见 hook manager 的集成方式。

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
- `post-merge`：合并后提醒或执行 index/doctor 检查。
- `post-checkout`：切换分支后提醒或执行上下文健康检查。

## Husky

Husky 项目可在 `.husky/pre-commit` 中显式调用：

```bash
docx update --staged
```

如果希望提交前生成 AI proposal：

```bash
docx update --staged --ai
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
