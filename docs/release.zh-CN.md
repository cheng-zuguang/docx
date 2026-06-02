# Release Workflow

本文档说明如何通过 GitHub Actions 发布 `docx` 的 GitHub Release 和 npm 包。

## 触发方式

发布 workflow 位于 `.github/workflows/release.yml`。

推送 `v*` tag 时会自动触发：

```bash
git tag v0.1.0
git push origin v0.1.0
```

workflow 会执行：

1. `go test ./...`
2. `npm pack --dry-run`
3. 构建 macOS、Linux、Windows 的 amd64/arm64 二进制
4. 上传 GitHub Release assets
5. 在满足条件时发布 npm 包

## GitHub Release Assets

安装脚本和 npm postinstall 依赖这些 asset 名称：

```text
docx_darwin_amd64.tar.gz
docx_darwin_arm64.tar.gz
docx_linux_amd64.tar.gz
docx_linux_arm64.tar.gz
docx_windows_amd64.zip
docx_windows_arm64.zip
```

## npm 发布

npm 发布由 workflow 中的 `npm` job 执行，但默认需要显式启用。当前 workflow 使用 npm Trusted Publishing，不依赖长期 `NPM_TOKEN`。

需要在 GitHub 仓库中配置：

- Variable: `PUBLISH_NPM=true`

还需要在 npm package 或 scope 的 Trusted Publishing 设置中绑定 GitHub Actions：

```text
Package: @chengzg/docx
Owner / organization: cheng-zuguang
Repository: docx
Workflow filename: release.yml
```

如果 npm 页面要求 workflow path，填写：

```text
.github/workflows/release.yml
```

如果 npm 页面要求 environment，而 workflow 没有配置 environment，则留空。

首次发布 scoped public package 时，workflow 使用：

```bash
npm publish --access public
```

## 发布前检查

发布 tag 前建议本地先执行：

```bash
go test ./...
npm pack --dry-run
```

并确认 `package.json` 版本与 tag 版本一致。
