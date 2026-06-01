# SFS — SmallFileSync

A terminal-based file sync tool powered by WebDAV, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- Sync files across machines via WebDAV (e.g. Jianguoyun)
- Chunk-based upload/download with MD5 verification
- Conflict detection between local and remote changes
- Auto sync with countdown timer
- i18n support (中文 / English)
- Migrates data from the legacy uTools plugin

## Install

```bash
go install github.com/vst93/sfs@latest
```

## Usage

```bash
sfs
```

Press `s` to configure WebDAV connection, then `a` to add files.

### Keybindings

| Key | Action |
|-----|--------|
| `j`/`k` | Navigate |
| `Enter` | Smart sync (auto decide direction) |
| `u` | Force upload |
| `d` | Force download |
| `x` | Delete record |
| `e` | Set local directory |
| `a` | Add file |
| `s` | Settings |
| `y` | Sync all |
| `o` | Toggle auto sync |
| `r` | Refresh |
| `Ctrl+Y` | Copy full file path |
| `L` | Toggle language |
| `?` | Help |
| `q` | Quit |

## Config

Settings are stored at `~/.config/small-filesync/settings.json`.

## License

MIT
