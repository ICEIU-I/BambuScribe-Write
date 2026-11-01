# macOS 发布

源码与正式下载统一在 [ICEIU-I/BambuScribe-Write](https://github.com/ICEIU-I/BambuScribe-Write)。支持 Apple Silicon、macOS 13+；应用使用临时签名，尚未经过 Apple 公证。安装方法见[安装说明](../releases/INSTALL.txt)。

## 构建

在 Apple Silicon Mac 安装 Go 1.25+、Node.js 22.12+、Xcode Command Line Tools 及 Wails v2.16.0。同步客户端 `wails.json`、`internal/buildinfo/version.go`、前端 `package.json` 和 `package-lock.json` 的应用版本，补充 `desktop/releases/版本.md` 和 AGENTS.md 验证记录后提交。

```sh
bash desktop/scripts/release-macos.sh 1.0.0 HEAD
cd desktop/build/releases/1.0.0
shasum -a 256 -c SHA256SUMS
```

脚本从指定 Git 提交创建干净源码副本，生成 DMG、ZIP、对应源码 ZIP、许可证、构建信息和校验文件。构建信息记录公开提交，不写入本机用户名、绝对路径或设备配置。输出目录已存在时拒绝覆盖；不会替换正在使用的应用。

## 发布

完成 Go 竞态检查、go vet、前端检查与交互回归，再验证安装包签名、版本、架构与校验值。使用对应公开提交创建同版本 Git 标签，确认附件只包含本版产物，再执行：

```sh
gh release create v1.0.0 --repo ICEIU-I/BambuScribe-Write --verify-tag --title 'BambuScribe Write 1.0.0' --notes-file desktop/build/releases/1.0.0/release-notes.md desktop/build/releases/1.0.0/*
```

公开前检查全部 Git 历史、提交作者与邮箱、源码压缩包及附件，移除个人项目、主目录路径、访问码、API Key、真实设备信息和运行日志。发布提交使用 GitHub 公开身份。安装包上传后再次下载并核对 SHA-256。

## 1.0.0 更新过渡

1. 主仓库发布 1.0.0，源码标签与 `build-info.json` 的 `sourceCommit` 一致。
2. 将完全相同的 1.0.0 附件发布到旧下载仓库 `ICEIU-I/BambuScribe-Write-Releases`。旧客户端会严格校验附件所在仓库，不能仅提供跳转链接。
3. 核验旧版能发现和下载旧仓库的 1.0.0，新版能从主仓库检查更新。
4. 更新旧仓库首页和源码入口，然后归档；保留过渡包供尚未升级的用户获取。

从 1.0.0 起只向主仓库发布新版。0.1.0 不支持应用内升级，需要手动安装。归档不删除旧下载文件。

## 历史版本

0.1.0–0.1.8 使用清理过个人信息的对应源码重新构建，修正旧帮助链接，安装包与原发布文件的 SHA-256 不同。每版 `build-info.json` 指向公开源码提交；源码、安装包和校验文件成套发布。原始仓库历史与附件保留在私有归档，不随公开仓库提供。
