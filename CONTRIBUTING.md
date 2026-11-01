# 开发与贡献

先阅读 [README](README.md) 的构建要求与 [AGENTS.md](AGENTS.md) 的协作规则。保持改动集中在对应模块，修复缺陷时加入能复现问题的回归检查。

在 macOS 安装 Go、Node.js、Xcode Command Line Tools 和 Wails v2.16.0 后执行：

```sh
cd desktop
wails build -platform darwin/arm64 -trimpath
go test -race ./internal/...
go vet ./...
cd frontend
npm run check
node --test tests/*.test.mjs
npx playwright install chromium webkit
npm run test:browser
```

Wails 构建生成的 `frontend/wailsjs`、`frontend/dist` 和 `build/bin` 不提交。浏览器测试使用模拟接口，不读取个人草稿或连接真实打印机。

提交前运行 `git diff --check`。界面改动使用 Prettier，Go 文件使用 gofmt。按功能提交，并在 AGENTS.md 追加简短改动和验证结果。

不要提交 API Key、访问码、设备序列号、真实网络地址、个人项目、绝对主目录路径、诊断文件或运行日志。文档使用示例路径，测试使用虚构数据。提交作者请使用自己的 GitHub 公开身份或 GitHub noreply 邮箱。

提交 Issue 时说明应用版本、macOS 版本、复现步骤和预期结果；分享截图或 `.ccf` 前先移除个人内容。涉及机械动作的改动需说明设备型号与验证范围，不把模拟测试描述为实机验收。
