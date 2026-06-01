# SFS — SmallFileSync

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg?logo=go)](https://golang.org)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg)](.)

<div align="center">

A WebDAV-based terminal file sync tool | 基于 WebDAV 的终端文件同步工具

</div>

---

## Table of Contents | 目录

- [Introduction | 项目介绍](#introduction--项目介绍)
- [Features | 功能特性](#features--功能特性)
- [Limits | 限制](#limits--限制)
- [Installation | 安装](#installation--安装)
- [Build from Source | 从源码构建](#build-from-source--从源码构建)
- [Quick Start | 快速开始](#quick-start--快速开始)
- [Configuration | 配置说明](#configuration--配置说明)
- [Configuration Details | 配置详情](#configuration-details--配置详情)
- [Keybindings | 快捷键](#keybindings--快捷键)
- [File Status | 文件状态](#file-status--文件状态)
- [Architecture | 架构设计](#architecture--架构设计)
- [Troubleshooting | 故障排查](#troubleshooting--故障排查)
- [Contributing | 参与贡献](#contributing--参与贡献)
- [Changelog | 更新日志](#changelog--更新日志)
- [License | 许可证](#license--许可证)

---

## Introduction | 项目介绍

SFS (SmallFileSync) is a terminal-based file synchronization tool built on the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework. With support for mainstream WebDAV providers like Jianguoyun (Nutstore), it's designed for developers who frequently sync small files across multiple machines.

SFS (SmallFileSync) 是一款基于 WebDAV 的终端文件同步工具，使用 Go 语言编写，搭配 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 构建的直观 TUI 界面。支持坚果云等主流 WebDAV 服务，专为需要在多台机器之间频繁同步小文件的开发者设计。

---

## Features | 功能特性

- **Whole-file Transfer** — Files are uploaded/downloaded as a single unit (no chunking)
- **MD5 Verification** — Integrity checks ensuring lossless transfers
- **Conflict Detection** — Smart conflict identification with manual resolution
- **Auto Sync** — Configurable automatic sync countdown
- **i18n** — Switch between Chinese and English UI with a single key (`L`)
- **Data Migration** — Import data from the legacy uTools plugin; old chunked records are automatically migrated to the new single-file format

- **整文件传输** — 文件作为单个单元上传/下载（不分块）
- **MD5 校验** — 完整性检查，确保无损传输
- **冲突检测** — 智能识别冲突，支持手动解决
- **自动同步** — 可配置的自动同步倒计时
- **国际化** — 一键切换中英文界面（`L` 键）
- **数据迁移** — 支持从旧版 uTools 插件导入数据；旧的分块记录会自动迁移到新的整文件格式

---

## Limits | 限制

| Item | Limit |
|------|-------|
| Single file size | ≤ 10MB |
| Sync files count | ≤ 30 |

| 项目 | 限制 |
|------|------|
| 单文件大小 | ≤ 10MB |
| 同步文件数 | ≤ 30 |

---

## Installation | 安装

```bash
go install github.com/vst93/sfs@latest
```

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg?logo=go)](https://golang.org)

Requires **Go 1.21+**

需要 **Go 1.21+**

---

## Build from Source | 从源码构建

**Prerequisites:** Go 1.21+, Git

**前置要求：** Go 1.21+、Git

```bash
git clone https://github.com/vst93/sfs.git
cd sfs
go build -o sfs
```

Optionally install to your `$GOPATH/bin`:

可选安装到 `$GOPATH/bin`：

```bash
go install
```

---

## Quick Start | 快速开始

```bash
sfs
```

1. Press `s` to configure your WebDAV connection (URL, username, password)
2. Press `e` to set local sync directory
3. Press `a` to add files for synchronization
4. Press `Enter` for smart sync, or `y` to sync all

1. 按 `s` 配置 WebDAV 连接（URL、用户名、密码）
2. 按 `e` 设置本地同步目录
3. 按 `a` 添加需要同步的文件
4. 按 `Enter` 智能同步，或按 `y` 同步全部

---

## Configuration | 配置说明

Configuration files are stored in `~/.config/small-filesync/`:

| File | Description |
|------|-------------|
| `settings.json` | WebDAV server config, auto-sync toggle, language |
| `dirmap_<uid>.json` | Local directory mappings (separated by machine UID) |
| `filestate_<uid>.json` | File sync states (MD5, mtime, last sync time) |
| `uid` | Machine unique identifier |

配置文件存储在 `~/.config/small-filesync/`：

| 文件 | 说明 |
|------|------|
| `settings.json` | WebDAV 服务器配置、自动同步开关、语言设置 |
| `dirmap_<uid>.json` | 本地目录映射（按机器 UID 分离） |
| `filestate_<uid>.json` | 文件同步状态（MD5、修改时间、上次同步时间） |
| `uid` | 机器唯一标识符 |

---

## Configuration Details | 配置详情

Complete `settings.json` example (Jianguoyun / Nutstore):

完整的 `settings.json` 示例（以坚果云为例）：

```json
{
  "autoSync": true,
  "language": "en",
  "storage": {
    "type": "webdav",
    "webdav": {
      "endpoint": "https://dav.jianguoyun.com/dav/",
      "username": "your-email@example.com",
      "password": "your-app-password",
      "basePath": "small-file-sync"
    }
  }
}
```

| Field | Description |
|-------|-------------|
| `storage.webdav.endpoint` | WebDAV server URL |
| `storage.webdav.username` | Account email |
| `storage.webdav.password` | App-specific password |
| `storage.webdav.basePath` | Remote directory (default: `small-file-sync`) |
| `autoSync` | Enable automatic sync (`true`/`false`) |
| `language` | UI language: `"en"` or `"zh"` |

| 字段 | 说明 |
|------|------|
| `storage.webdav.endpoint` | WebDAV 服务器地址 |
| `storage.webdav.username` | 登录邮箱 |
| `storage.webdav.password` | 应用专用密码 |
| `storage.webdav.basePath` | 远程目录（默认 `small-file-sync`） |
| `autoSync` | 是否启用自动同步 |
| `language` | 界面语言：`"en"` 或 `"zh"` |

**Config file path by platform:**

**各平台配置文件路径：**

| Platform | Path |
|----------|------|
| Linux | `~/.config/small-filesync/` |
| macOS | `~/Library/Application Support/small-filesync/` |
| Windows | `%APPDATA%\small-filesync\` |

| 平台 | 路径 |
|------|------|
| Linux | `~/.config/small-filesync/` |
| macOS | `~/Library/Application Support/small-filesync/` |
| Windows | `%APPDATA%\small-filesync\` |

---

## Keybindings | 快捷键

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `PgUp` / `PgDn` | Page up/down |
| `g` / `G` | Jump to first/last |
| `Enter` | Smart sync (auto-detect direction) |
| `u` | Force upload |
| `d` | Force download |
| `x` | Delete sync record |
| `e` | Set local directory |
| `a` | Add file |
| `s` | Storage settings |
| `y` | Sync all |
| `o` | Toggle auto sync |
| `r` | Refresh |
| `Ctrl+Y` | Copy file path |
| `L` | Switch language |
| `?` | Help |
| `q` | Quit |

| 按键 | 操作 |
|------|------|
| `j` / `k` | 上/下导航 |
| `PgUp` / `PgDn` | 上/下翻页 |
| `g` / `G` | 跳转到首/末项 |
| `Enter` | 智能同步（自动检测方向） |
| `u` | 强制上传 |
| `d` | 强制下载 |
| `x` | 删除同步记录 |
| `e` | 设置本地目录 |
| `a` | 添加文件 |
| `s` | 存储设置 |
| `y` | 同步全部 |
| `o` | 切换自动同步 |
| `r` | 刷新 |
| `Ctrl+Y` | 复制文件路径 |
| `L` | 切换语言 |
| `?` | 帮助 |
| `q` | 退出 |

---

## File Status | 文件状态

| Status | Description |
|--------|-------------|
| Synced | Local and cloud match |
| To Upload | Local has changes |
| To Download | Cloud has changes |
| First Upload | Not yet uploaded to cloud |
| Missing | Local file absent, can restore from cloud |
| Conflict | Both sides modified, manual resolution needed |
| Unbound | No local directory linked |

| 状态 | 说明 |
|------|------|
| Synced | 本地与云端一致 |
| To Upload | 本地有变更 |
| To Download | 云端有变更 |
| First Upload | 尚未上传到云端 |
| Missing | 本地文件缺失，可从云端恢复 |
| Conflict | 双方均有修改，需手动解决 |
| Unbound | 未绑定本地目录 |

---

## Architecture | 架构设计

```
+------------------------------------------+
|           SFS TUI Interface              |
|        (Bubble Tea / Lipgloss)           |
+------------------------------------------+
|            Sync Engine                   |
|  +-------------+  +-------------------+  |
|  | Whole-file  |  |  MD5 Check /      |  |
|  | Transfer    |  |  Conflict Detect  |  |
|  +-------------+  +-------------------+  |
+------------------------------------------+
|           Storage Layer                  |
|  +-------------+  +-------------------+  |
|  |   WebDAV    |  |  Local JSON Files |  |
|  |  (Remote)   |  |  (State / Config) |  |
|  +-------------+  +-------------------+  |
+------------------------------------------+
```

**Remote storage layout:**

**远程存储布局：**

    <basePath>/
      meta/
        fileList.json      <- File list (FileRecord[])
      data/
        file_<id>          <- Whole file data (Base64 encoded)

---

## Troubleshooting | 故障排查

**WebDAV connection failed**

- Verify your `webdav_url` is correct and accessible from your network
- Ensure your username and password are valid (use an app-specific password if required by the provider)
- Check firewall or proxy settings that may block HTTPS requests

**WebDAV 连接失败**

- 确认 `webdav_url` 正确且可从当前网络访问
- 确保用户名和密码有效（部分服务提供商要求使用应用专用密码）
- 检查防火墙或代理设置是否阻止了 HTTPS 请求

**Sync timeout**

- Large files or slow networks may cause timeouts; try syncing fewer files at a time
- Verify network stability and consider increasing the connection timeout if your provider supports it

**同步超时**

- 大文件或网络较慢可能导致超时；尝试减少单次同步的文件数量
- 检查网络稳定性，如服务提供商支持，可考虑增加连接超时时间

**Conflict resolution**

- When a file shows `Conflict` status, both local and remote versions have been modified
- Use `u` to force upload (keep local) or `d` to force download (keep remote)
- Review both versions manually before choosing to avoid data loss

**冲突解决**

- 文件显示 `Conflict` 状态时，表示本地和远程版本均已被修改
- 按 `u` 强制上传（保留本地版本）或按 `d` 强制下载（保留远程版本）
- 选择前请手动检查两个版本，避免数据丢失

**Old data migration (FileIds to FileID)**

- SFS automatically migrates legacy chunked data (FileIds) to the new single-file format (FileID) on first access
- No manual action is required; migration runs in the background when you open the TUI
- If migration fails, check the error message and ensure sufficient local disk space is available

**旧数据迁移（FileIds 到 FileID）**

- SFS 会在首次访问时自动将旧版分块数据（FileIds）迁移到新的整文件格式（FileID）
- 无需手动操作；打开 TUI 时迁移会在后台自动运行
- 如果迁移失败，请查看错误信息并确保本地磁盘空间充足

---

## Contributing | 参与贡献

Contributions are welcome! Here's how you can help:

- **Bug reports** — Open an issue with steps to reproduce, expected behavior, and actual behavior
- **Pull requests** — Fork the repo, create a feature branch, and submit a PR with a clear description
- **Code style** — Run `gofmt` and `go vet` before submitting; ensure no warnings
- **i18n** — Translation contributions for additional languages are appreciated

欢迎参与贡献！以下是参与方式：

- **问题报告** — 提交 Issue 时请附上复现步骤、预期行为和实际行为
- **代码贡献** — Fork 仓库，创建功能分支，提交 PR 并附上清晰的说明
- **代码规范** — 提交前请运行 `gofmt` 和 `go vet`，确保无警告
- **国际化** — 欢迎贡献其他语言的翻译

---

## Changelog | 更新日志

**v0.1.0**

- Initial release
- WebDAV-based sync engine
- Whole-file transfer with MD5 verification
- Internationalization (Chinese / English)
- Automatic sync with configurable interval

**v0.1.0**

- 首次发布
- 基于 WebDAV 的同步引擎
- 整文件传输与 MD5 校验
- 国际化支持（中文 / 英文）
- 可配置间隔的自动同步

---

## License | 许可证

MIT License © 2026 [vst93](https://github.com/vst93)
