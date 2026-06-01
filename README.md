# SFS — SmallFileSync

> A WebDAV-based terminal file sync tool / 基于 WebDAV 的终端文件同步工具

---

## English

### Introduction

SFS (SmallFileSync) is a terminal-based file synchronization tool built on the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework. With support for mainstream WebDAV providers like Jianguoyun (Nutstore), it's designed for developers who frequently sync small files across multiple machines.

### Features

- Upload/download with MD5 verification — integrity checks ensuring lossless transfers
- Conflict detection — smart conflict identification with manual resolution
- Auto sync — configurable automatic sync countdown
- Data migration — import data from the legacy uTools plugin

### Limits

| Item | Limit |
|------|-------|
| Single file size | ≤ 10MB |
| Sync files count | ≤ 30 |

### Installation

```bash
go install github.com/vst93/sfs@latest
```

Requires **Go 1.21+**

### Quick Start

```bash
sfs
```

1. Press `s` to configure your WebDAV connection (URL, username, password)
2. Press `e` to set local sync directory
3. Press `a` to add files for synchronization
4. Press `Enter` for smart sync, or `y` to sync all

### Configuration

Configuration files are stored in `~/.config/small-filesync/`:

| File | Description |
|------|-------------|
| `settings.json` | WebDAV server config, auto-sync toggle, language |
| `dirmap_<uid>.json` | Local directory mappings (separated by machine UID) |
| `filestate_<uid>.json` | File sync states (MD5, mtime, last sync time) |
| `uid` | Machine unique identifier |

### Keybindings

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

### File Status

| Status | Description |
|--------|-------------|
| Synced | Local and cloud match |
| To Upload | Local has changes |
| To Download | Cloud has changes |
| First Upload | Not yet uploaded to cloud |
| Missing | Local file absent, can restore from cloud |
| Conflict | Both sides modified, manual resolution needed |
| Unbound | No local directory linked |

### Architecture

```
┌──────────────────────────────────────────┐
│              SFS TUI Interface           │
│         (Bubble Tea / Lipgloss)          │
├──────────────────────────────────────────┤
│             Sync Engine                  │
│  ┌─────────────────────────────────────┐ │
│  │  MD5 Check /  Conflict Detection   │ │
│  └─────────────────────────────────────┘ │
├──────────────────────────────────────────┤
│            Storage Layer                 │
│  ┌───────────┐  ┌─────────────────────┐  │
│  │  WebDAV   │  │   Local JSON Files  │  │
│  │ (Remote)  │  │   (State / Config)  │  │
│  └───────────┘  └─────────────────────┘  │
└──────────────────────────────────────────┘
```

**Remote storage layout:**
```
<basePath>/
  meta/
    fileList.json      ← File list (FileRecord[])
  data/
    file_<id>          ← File data (Base64 encoded)
```

---

## 中文

### 项目简介

SFS (SmallFileSync) 是一款基于 WebDAV 协议的终端文件同步工具，采用 Go 语言编写，基于 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 构建直观的 TUI 界面。支持坚果云 (Jianguoyun) 等主流 WebDAV 服务，专为需要频繁在多台机器间同步小文件的开发者设计。

### 特性

- 基于 MD5 校验的上传下载 — 完整性验证，确保文件无损传输
- 冲突检测 — 智能识别本地与远程的文件冲突，支持手动解决
- 自动同步 — 可配置的自动同步倒计时
- 数据迁移 — 支持从旧版 uTools 插件无缝迁移数据

### 限制

| 项目 | 限制 |
|------|------|
| 单文件大小 | ≤ 10MB |
| 同步文件数 | ≤ 30 个 |

### 安装

```bash
go install github.com/vst93/sfs@latest
```

要求 **Go 1.21+**

### 快速开始

```bash
sfs
```

1. 按 `s` 配置 WebDAV 连接（地址、用户名、密码）
2. 按 `e` 设置本地同步目录
3. 按 `a` 添加需要同步的文件
4. 按 `Enter` 智能同步，或按 `y` 全部同步

### 配置说明

配置文件存储于 `~/.config/small-filesync/` 目录：

| 文件 | 说明 |
|------|------|
| `settings.json` | WebDAV 服务器配置 + 自动同步开关 + 语言偏好 |
| `dirmap_<uid>.json` | 本地目录映射关系（按机器 UID 分离） |
| `filestate_<uid>.json` | 文件同步状态记录（MD5、mtime、上次同步时间） |
| `uid` | 本机唯一标识符 |

### 快捷键

| 按键 | 功能 |
|------|------|
| `j` / `k` | 上/下导航 |
| `PgUp` / `PgDn` | 上/下翻页 |
| `g` / `G` | 跳到首/尾 |
| `Enter` | 智能同步（自动判断方向） |
| `u` | 强制上传 |
| `d` | 强制下载 |
| `x` | 删除同步记录 |
| `e` | 设置本地目录 |
| `a` | 添加文件 |
| `s` | 存储设置 |
| `y` | 全部同步 |
| `o` | 开关自动同步 |
| `r` | 刷新 |
| `Ctrl+Y` | 复制文件路径 |
| `L` | 切换语言 |
| `?` | 帮助 |
| `q` | 退出 |

### 文件状态

| 状态 | 说明 |
|------|------|
| 已同步 | 本地与云端一致 |
| 待上传 | 本地有新内容 |
| 待下载 | 云端有新内容 |
| 首次上传 | 尚未上传到云端 |
| 本地缺失 | 本地文件不存在，可从云端恢复 |
| 冲突 | 本地与云端均已修改，需手动选择 |
| 未关联 | 未绑定本地目录 |

### 技术架构

```
┌──────────────────────────────────────────┐
│              SFS TUI 界面                │
│         (Bubble Tea / Lipgloss)          │
├──────────────────────────────────────────┤
│              同步引擎                     │
│  ┌─────────────────────────────────────┐ │
│  │  MD5 校验 / 冲突检测                 │ │
│  └─────────────────────────────────────┘ │
├──────────────────────────────────────────┤
│              存储层                       │
│  ┌───────────┐  ┌─────────────────────┐  │
│  │  WebDAV   │  │   本地 JSON 文件    │  │
│  │  (远程)    │  │   (状态 / 配置)     │  │
│  └───────────┘  └─────────────────────┘  │
└──────────────────────────────────────────┘
```

**远程存储结构：**
```
<basePath>/
  meta/
    fileList.json      ← 文件清单 (FileRecord[])
  data/
    file_<id>          ← 文件数据 (Base64 编码)
```

---

## License / 许可证

MIT License © 2026 [vst93](https://github.com/vst93)
