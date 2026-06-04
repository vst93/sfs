# SFS — SmallFileSync

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg?logo=go)](https://golang.org)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows%20%7C%20Android-lightgrey.svg)](.)

<div align="center">

A WebDAV-based file sync tool with terminal (TUI) and web interface

**[English](#english)** · **[中文](#中文)**

<img src="https://raw.githubusercontent.com/vst93/sfs/main/image.png" alt="SFS Screenshot" width="800" />

</div>

---

## English

- [Introduction](#introduction)
- [Features](#features)
- [Why Not Just Use Cloud Storage?](#why-not-just-use-cloud-storage)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Keybindings](#keybindings)
- [File Status](#file-status)
- [Architecture](#architecture)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

### Introduction

SFS (SmallFileSync) is a file sync tool built on WebDAV, supporting both a **terminal TUI** ([Bubble Tea](https://github.com/charmbracelet/bubbletea)) and a **web interface**. It syncs individual files via WebDAV (Jianguoyun/Nutstore, etc.), designed for developers who need small files — configs, dotfiles, IDE settings — consistent across machines.

- **Terminal mode (`sfs`)** — Interactive TUI in your terminal, full keyboard-driven workflow
- **Web mode (`sfs web [port]`)** — Browser-based UI, auto-opens in your default browser

### Features

- **Whole-file Transfer** — No chunking; files uploaded/downloaded as a unit
- **MD5 Verification** — Integrity checks for lossless transfers
- **Conflict Detection** — Smart detection with manual resolution
- **Auto Sync** — Configurable automatic sync countdown
- **Dual Interface** — Terminal TUI + Web UI, choose your preferred workflow
- **i18n** — Switch Chinese/English with `L` key (terminal) or browser language
- **Config Export/Import** — Export settings as a shareable CLI command (`Ctrl+B` in settings), import via `sfs --import-config <base64>` on another machine
- **Data Migration** — Auto-migrates legacy uTools plugin data
- **Platform** — Linux, macOS, Windows, Android (via Termux)

> Single file size limit: ≤ 200MB. No hard limit on file count, but keep sizes modest for best performance.

### Why Not Just Use Cloud Storage?

| | Cloud Storage | SFS |
|--|--|--|
| **Sync approach** | Watches entire folders | Sync individual chosen files |
| **Storage usage** | Files in cloud, consuming quota | Uses your existing WebDAV; files stay local |
| **Conflict handling** | Auto-creates duplicate files | Detects & prompts manual resolution |
| **Setup** | Install client, point to folder | Add files one by one, no daemon |

### Installation

**Homebrew (macOS/Linux):**

```bash
brew install vst93/tap/sfs
```

**Install script:**

```bash
curl -fsSL https://github.com/vst93/sfs/releases/latest/download/install.sh | bash
```

Options: `--install-dir /custom/path`, `FORCE_INSTALL=1`, `AUTO_DELETE_INSTALL_SCRIPT=0`

**go install:**

```bash
go install github.com/vst93/sfs@latest
```

**Build from source:**

```bash
git clone https://github.com/vst93/sfs.git && cd sfs
go build -o sfs
```

### Quick Start

**Terminal mode (default):**

```bash
sfs
```

1. `s` — Configure WebDAV connection (URL, username, password)
2. `e` — Set local sync directory
3. `a` — Add files for syncing
4. `Enter` — Smart sync (auto-detect direction), or `y` to sync all

**Web mode:**

```bash
sfs web        # Start on default port 8080
sfs web 3000   # Start on custom port
```

Then open the displayed URL in your browser to manage files via the web interface.

### Configuration

Config files stored in `~/.config/small-filesync/` (Linux), `~/Library/Application Support/small-filesync/` (macOS), `%APPDATA%\small-filesync\` (Windows):

| File | Description |
|------|-------------|
| `settings.json` | WebDAV config, auto-sync toggle, language |
| `dirmap_<uid>.json` | Local directory mappings (per machine) |
| `filestate_<uid>.json` | File sync states (MD5, mtime) |
| `uid` | Machine unique identifier |
| `export-command.txt` | Exported import command (generated on `Ctrl+B`) |

`settings.json` example (Jianguoyun):

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

### Keybindings

| Key | Action | Key | Action |
|-----|--------|-----|--------|
| `j` / `k` | Up / Down | `g` / `G` | First / Last |
| `Enter` | Smart sync | `y` | Sync all |
| `u` | Force upload | `d` | Force download |
| `a` | Add file | `x` | Delete record |
| `s` | Storage settings | `e` | Set local directory |
| `o` | Toggle auto sync | `r` | Refresh |
| `Ctrl+Y` | Copy file path | `L` | Switch language |
| `?` | Help | `q` | Quit |

**Settings view:**

| Key | Action |
|-----|--------|
| `Ctrl+B` | Export config (copies CLI command to clipboard, also saved to `~/.config/small-filesync/export-command.txt`) |
| `t` | Test WebDAV connection |
| `p` | Toggle password visibility |

**Config import:**

```bash
sfs --import-config <base64-encoded-json>
```

The command will decode and display the configuration details (password masked), prompt for confirmation, and save to `~/.config/small-filesync/settings.json` on approval.

### File Status

| Status | Description |
|--------|-------------|
| Synced | Local and cloud match |
| To Upload | Local has changes |
| To Download | Cloud has changes |
| First Upload | Not yet uploaded |
| Missing | Local absent; restorable from cloud |
| Conflict | Both modified; manual resolution needed |
| Unbound | No local directory linked |

### Architecture

```
┌─────────────────────────────────┐
│         SFS UI Layer            │
│  ┌───────────┐ ┌─────────────┐  │
│  │ Terminal   │ │ Web (SPA)   │  │
│  │ TUI        │ │ HTTP Server │  │
│  └─────┬─────┘ └──────┬──────┘  │
│        └───────┬───────┘         │
│          Sync Engine             │
│  (Whole-file Transfer · MD5      │
│   Check · Conflict Detect)       │
│                │                 │
│          Storage Layer           │
│  (WebDAV Remote + Local JSON)    │
└─────────────────────────────────┘
```

**Run modes:**

| Command | Mode | Description |
|---------|------|-------------|
| `sfs` | Terminal | Interactive TUI (Bubble Tea / Lipgloss) |
| `sfs web [port]` | Web | Browser-based UI (default port 8080) |
| `sfs --import-config <base64>` | CLI | Import configuration from base64-encoded JSON |

Remote storage layout:

    <basePath>/
      meta/fileList.json      ← File list
      data/file_<id>          ← Whole file (Base64)

### Troubleshooting

**Connection failed** — Verify URL, credentials (use app-specific password for Jianguoyun), and check firewall/proxy.

**Sync timeout** — Reduce batch size; check network stability.

**Conflict** — File shows `Conflict` when both sides modified. Use `u` (keep local) or `d` (keep remote).

**Legacy migration** — Runs automatically on first launch. No manual action needed.

### Contributing

Bug reports, PRs, and i18n contributions are welcome. Run `gofmt` and `go vet` before submitting.

### License

MIT License © 2026 [vst](https://github.com/vst93)

---

## 中文

- [项目介绍](#项目介绍)
- [功能特性](#功能特性)
- [为什么不直接用网盘？](#为什么不直接用网盘)
- [安装](#安装)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [快捷键](#快捷键)
- [文件状态](#文件状态)
- [架构设计](#架构设计)
- [故障排查](#故障排查)
- [参与贡献](#参与贡献)
- [许可证](#许可证)

### 项目介绍

SFS (SmallFileSync) 是一款基于 WebDAV 的文件同步工具，使用 Go 构建，同时支持**终端 TUI**（[Bubble Tea](https://github.com/charmbracelet/bubbletea)）和 **Web 浏览器界面**。支持坚果云等主流 WebDAV 服务，专为需要在多台机器间保持小文件（配置文件、dotfiles、IDE 设置）一致性的开发者设计。

- **终端模式 (`sfs`)** — 终端中的交互式 TUI，全键盘操作
- **Web 模式 (`sfs web [port]`)** — 浏览器界面，启动后自动打开默认浏览器

### 功能特性

- **整文件传输** — 文件作为单个单元上传/下载，不分块
- **MD5 校验** — 完整性检查，确保无损传输
- **冲突检测** — 智能识别冲突，支持手动解决
- **自动同步** — 可配置的自动同步倒计时
- **双界面** — 终端 TUI + Web 浏览器界面，自由选择使用方式
- **国际化** — `L` 键一键切换中英文界面
- **配置导入导出** — 在设置页按 `Ctrl+B` 导出配置为可分享的 CLI 命令，其他机器通过 `sfs --import-config <base64>` 导入
- **数据迁移** — 自动迁移旧版 uTools 插件数据
- **多平台** — Linux、macOS、Windows、Android（通过 Termux）

> 单文件大小限制：≤ 200MB。文件数量无硬性限制，但建议保持文件体积适中以获得最佳性能。

### 为什么不直接用网盘？

| | 网盘 | SFS |
|--|--|--|
| **同步方式** | 监听整个文件夹变更 | 按需同步手动指定的文件 |
| **存储占用** | 文件常驻云端，消耗配额 | 复用已有 WebDAV 存储 |
| **冲突处理** | 自动生成副本文件 | 检测并提示手动解决 |
| **配置** | 安装客户端，指向目录 | 逐个添加文件，无需守护进程 |

### 安装

**Homebrew (macOS/Linux)：**

```bash
brew install vst93/tap/sfs
```

**安装脚本：**

```bash
curl -fsSL https://github.com/vst93/sfs/releases/latest/download/install.sh | bash
```

可选参数：`--install-dir /自定义路径`、`FORCE_INSTALL=1`、`AUTO_DELETE_INSTALL_SCRIPT=0`

**go install：**

```bash
go install github.com/vst93/sfs@latest
```

**从源码构建：**

```bash
git clone https://github.com/vst93/sfs.git && cd sfs
go build -o sfs
```

### 快速开始

**终端模式（默认）：**

```bash
sfs
```

1. `s` — 配置 WebDAV 连接（URL、用户名、密码）
2. `e` — 设置本地同步目录
3. `a` — 添加需要同步的文件
4. `Enter` — 智能同步（自动检测方向），或 `y` 同步全部

**Web 模式：**

```bash
sfs web        # 使用默认端口 8080 启动
sfs web 3000   # 指定自定义端口
```

启动后自动打开浏览器，通过 Web 界面管理文件同步。

### 配置说明

配置文件路径：`~/.config/small-filesync/`（Linux）、`~/Library/Application Support/small-filesync/`（macOS）、`%APPDATA%\small-filesync\`（Windows）：

| 文件 | 说明 |
|------|------|
| `settings.json` | WebDAV 配置、自动同步、语言设置 |
| `dirmap_<uid>.json` | 本地目录映射（按机器 UID 分离） |
| `filestate_<uid>.json` | 文件同步状态（MD5、修改时间） |
| `uid` | 机器唯一标识符 |
| `export-command.txt` | 导出的导入命令（按 `Ctrl+B` 生成） |

`settings.json` 完整示例（以坚果云为例）：

```json
{
  "autoSync": true,
  "language": "zh",
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

### 快捷键

| 按键 | 操作 | 按键 | 操作 |
|------|------|------|------|
| `j` / `k` | 上/下导航 | `g` / `G` | 跳转首/末项 |
| `Enter` | 智能同步 | `y` | 同步全部 |
| `u` | 强制上传 | `d` | 强制下载 |
| `a` | 添加文件 | `x` | 删除记录 |
| `s` | 存储设置 | `e` | 设置本地目录 |
| `o` | 切换自动同步 | `r` | 刷新 |
| `Ctrl+Y` | 复制文件路径 | `L` | 切换语言 |
| `?` | 帮助 | `q` | 退出 |

**设置页：**

| 按键 | 操作 |
|------|------|
| `Ctrl+B` | 导出配置（CLI 命令自动复制到剪贴板，同时保存到 `~/.config/small-filesync/export-command.txt`） |
| `t` | 测试 WebDAV 连接 |
| `p` | 显示/隐藏密码 |

**配置导入：**

```bash
sfs --import-config <base64-编码的-json>
```

命令会解码并展示配置详情（密码掩码显示），确认后写入 `~/.config/small-filesync/settings.json`。

### 文件状态

| 状态 | 说明 |
|------|------|
| Synced | 本地与云端一致 |
| To Upload | 本地有变更 |
| To Download | 云端有变更 |
| First Upload | 尚未上传 |
| Missing | 本地缺失，可从云端恢复 |
| Conflict | 双方均有修改，需手动解决 |
| Unbound | 未绑定本地目录 |

### 架构设计

```
┌─────────────────────────────────┐
│          SFS 界面层             │
│  ┌───────────┐ ┌─────────────┐  │
│  │ 终端 TUI  │ │ Web (SPA)   │  │
│  │ Bubble Tea│ │ HTTP Server │  │
│  └─────┬─────┘ └──────┬──────┘  │
│        └───────┬───────┘         │
│          同步引擎               │
│  (整文件传输 · MD5 校验          │
│   · 冲突检测)                   │
│                │                 │
│          存储层                 │
│  (WebDAV 远程 + 本地 JSON)      │
└─────────────────────────────────┘
```

**运行模式：**

| 命令 | 模式 | 说明 |
|------|------|------|
| `sfs` | 终端 | 交互式 TUI (Bubble Tea / Lipgloss) |
| `sfs web [port]` | Web | 浏览器界面（默认端口 8080） |
| `sfs --import-config <base64>` | CLI | 从 base64 编码的 JSON 导入配置 |

远程存储布局：

    <basePath>/
      meta/fileList.json      ← 文件列表
      data/file_<id>          ← 整文件数据 (Base64)

### 故障排查

**连接失败** — 确认 URL 和凭据正确（坚果云需使用应用专用密码），检查防火墙/代理设置。

**同步超时** — 减少单次同步文件数量，检查网络稳定性。

**冲突** — 本地和远程均被修改时显示 `Conflict`，按 `u`（保留本地）或 `d`（保留远程）解决。

**旧数据迁移** — 首次启动自动执行，无需手动操作。

### 参与贡献

欢迎提交 Issue、PR 和翻译贡献。提交前请运行 `gofmt` 和 `go vet`。

### 许可证

MIT License © 2026 [vst](https://github.com/vst93)
