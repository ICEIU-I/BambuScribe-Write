# BambuScribe Write

面向 macOS Apple Silicon 的拓竹 A1 手写助手：在纸面编辑文字、公式、表格与图片，生成 G-code，并通过局域网上传到打印机。

[下载 1.0.1](https://github.com/ICEIU-I/BambuScribe-Write/releases/tag/v1.0.1) · [操作说明](desktop/README.md) · [输入规则](desktop/docs/writing-syntax.md) · [反馈问题](https://github.com/ICEIU-I/BambuScribe-Write/issues)

## 安装与使用

需要 M 系列 Mac、macOS 13 或更新版本。打开 DMG，将应用拖入 Applications；也可解压 ZIP 安装。应用使用临时签名，尚未经过 Apple 公证，首次打开方法见[安装说明](desktop/releases/INSTALL.txt)。

字库、排版、预览及系统 OCR 可离线使用；智能导入需要自行配置模型服务和 API Key。个人项目、打印机访问码和模型配置只保存在本机，不随源码或安装包提供。

连接 A1 前，请安装笔架、开启 LAN 模式及固件支持的开发者模式，完成归零、笔尖定位与抬落笔高度校准。先用短句试写；缺字或坐标越界会阻止输出。客户端负责上传，书写的启动、暂停、停止和换纸在 A1 端操作。

## 功能

- 纸面正文编辑、自动分页、页面排序、浅深色主题和手写变化。
- 中文手写字库，正文上下标、分式、根号与上横线。
- 可编辑表格和文本框，支持合并、拖动、缩放、复制与撤销。
- 图片线稿、擦除，以及可选的表格、图像和数据处理智能导入。
- `.ccf` 项目保存、逐页 G-code 导出、批量上传和多设备连接。
- [CLI](desktop/docs/cli.md) 编辑、校验、预览、导出和设备调用。

## 开发

需要 Go 1.25+、Node.js 22.12+、Xcode Command Line Tools。首次运行：

```sh
git clone https://github.com/ICEIU-I/BambuScribe-Write.git
cd BambuScribe-Write
go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0
export PATH="$(go env GOPATH)/bin:$PATH"
cd desktop
wails dev
```

构建应用：

```sh
cd desktop
wails build -platform darwin/arm64 -trimpath
```

构建会安装前端依赖、生成 Wails 绑定并编译前端。产物为 `desktop/build/bin/BambuScribe Write.app`。完整检查及贡献流程见 [CONTRIBUTING.md](CONTRIBUTING.md)，发布步骤见[发布文档](desktop/docs/releases.md)。

## 目录

| 目录                                  | 内容                            |
| ------------------------------------- | ------------------------------- |
| `desktop/frontend`                    | Vue 界面、状态与交互测试        |
| `desktop/internal/engine`             | 字库、排版、公式、图像与 G-code |
| `desktop/internal/application`        | 桌面接口与文件操作              |
| `desktop/internal/printer`            | 发现、MQTT、FTPS 与上传         |
| `desktop/internal/project`、`storage` | 项目格式与本地配置              |
| `desktop/internal/updater`            | 版本检查与安装更新              |
| `desktop/cmd`、`tools`                | CLI 与开发工具                  |
| `desktop/docs`、`releases`、`scripts` | 使用说明、发布记录与打包脚本    |

从 1.0.0 起，源码、安装包和问题反馈统一在本仓库。旧下载仓库只保留历史文件和 1.0.0 过渡包；0.1.1–0.1.8 可在应用内检查更新，升级后自动使用本仓库。0.1.0 请手动安装。

## 许可与来源

本项目基于 [Animesh-Varma/BambuScribe](https://github.com/Animesh-Varma/BambuScribe)，按 [GPL-3.0](LICENSE) 分发。内置“教案专用”和“DC体”由 [ICEIU-I](https://github.com/ICEIU-I) 原创制作，按同一许可证分发；依赖和来源见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。每个版本的完整源码可在对应发布页下载。
