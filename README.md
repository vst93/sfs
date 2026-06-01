# SFS — SmallFileSync

<div align="center">

A WebDAV-based terminal file sync tool | 基于 WebDAV 的终端文件同步工具

</div>

---

## English

### Introduction

SFS (SmallFileSync) is a terminal-based file synchronization tool built on the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework. With support for mainstream WebDAV providers like Jianguoyun (Nutstore), it's designed for developers who frequently sync small files across multiple machines.

### Features

- **Whole-file Transfer** — Files are uploaded/downloaded as a single unit (no chunking)
- **MD5 Verification** — Integrity checks ensuring lossless transfers
- **Conflict Detection** — Smart conflict identification with manual resolution
- **Auto Sync** — Configurable automatic sync countdown
- **i18n** — Switch between Chinese and English UI with a single key (`L`)
- **Data Migration** — Import data from the legacy uTools plugin; old chunked records are automatically migrated to the new single-file format

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
```
<basePath>/
  meta/
    fileList.json      <- File list (FileRecord[])
  data/
    file_<id>          <- Whole file data (Base64 encoded)
```

---

## Chinese

### Project Introduction

SFS (SmallFileSync) is a terminal-based file synchronization tool built on WebDAV, written in Go with an intuitive TUI powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea). It supports mainstream WebDAV providers like Jianguoyun (Nutstore) and is designed for developers who frequently sync small files across multiple machines.

### Features

- **Whole-file Transfer** — Files are uploaded/downloaded as a single unit (no chunking)
- **MD5 Verification** — Integrity checks ensuring lossless transfers
- **Conflict Detection** — Smart conflict identification with manual resolution
- **Auto Sync** — Configurable automatic sync countdown
- **i18n** — Switch between Chinese and English UI with a single key (`L`)
- **Data Migration** — Import data from the legacy uTools plugin; old chunked records are automatically migrated to the new single-file format

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
```
<basePath>/
  meta/
    fileList.json      <- File list (FileRecord[])
  data/
    file_<id>          <- Whole file data (Base64 encoded)
```

---

## License

MIT License © 2026 [vst93](https://github.com/vst93)
