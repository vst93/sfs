package i18n

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Locale represents a supported language.
type Locale string

const (
	Zh Locale = "zh"
	En Locale = "en"
)

var currentLocale Locale = Zh

// SetLocale sets the active locale.
func SetLocale(l Locale) {
	currentLocale = l
}

// GetLocale returns the active locale.
func GetLocale() Locale {
	return currentLocale
}

// LocaleName returns the display name of a locale.
func LocaleName(l Locale) string {
	switch l {
	case Zh:
		return "中文"
	case En:
		return "English"
	default:
		return string(l)
	}
}

// ToggleLocale switches to the next locale and returns the new one.
func ToggleLocale() Locale {
	switch currentLocale {
	case Zh:
		currentLocale = En
	case En:
		currentLocale = Zh
	default:
		currentLocale = Zh
	}
	return currentLocale
}

// AvailableLocales returns all supported locales.
func AvailableLocales() []Locale {
	return []Locale{Zh, En}
}

// DetectLocale detects the system locale from environment variables.
func DetectLocale() Locale {
	for _, env := range []string{"SFS_LANG", "LC_ALL", "LC_MESSAGES", "LANG"} {
		val := os.Getenv(env)
		if val != "" {
			lower := strings.ToLower(val)
			if strings.HasPrefix(lower, "en") {
				return En
			}
			if strings.HasPrefix(lower, "zh") {
				return Zh
			}
			// Unrecognized locale defaults to English
			return En
		}
	}
	// macOS: try to detect from system defaults
	if runtime.GOOS == "darwin" {
		return Zh
	}
	return En
}

// T returns the translated string for the given key.
// Optional args are passed through fmt.Sprintf if the translation contains format verbs.
func T(key string, args ...any) string {
	m, ok := translations[currentLocale]
	if !ok {
		m = translations[Zh]
	}
	format, ok := m[key]
	if !ok {
		// Fallback to zh
		if m2, ok2 := translations[Zh]; ok2 {
			if f, ok3 := m2[key]; ok3 {
				format = f
			}
		}
		if format == "" {
			return key
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(format, args...)
	}
	return format
}

var translations = map[Locale]map[string]string{
	Zh: {
		// ── Common ──────────────────────────────────────────────────────────
		"common.success": "成功",
		"common.failure": "失败",
		"common.error":   "错误",
		"common.warning": "注意",
		"common.cancel":  "取消",
		"common.close":   "关闭",

		// ── Toast ───────────────────────────────────────────────────────────
		"toast.copied":      "已复制到剪贴板",
		"toast.copied_path": "已复制路径: %s",
		"toast.no_dir":      "该文件尚未关联本地目录",

		// ── File status keys (used in fileLine and computeFileState) ────────
		"status.matched":        "已同步",
		"status.pending_upload": "待上传",
		"status.download":       "待下载",
		"status.initial_upload": "首次上传",
		"status.missing":        "本地缺失",
		"status.conflict":       "冲突",
		"status.unbound":        "未关联",

		// ── File status details (shown in detail line) ──────────────────────
		"status.unbound.detail":        "请先为当前设备设置本地目录",
		"status.initial_upload.detail": "请执行首次上传",
		"status.missing.detail":        "本地文件缺失或不可读，可从云端恢复",
		"status.matched.detail":        "本地与云端一致",
		"status.conflict.detail":       "本地与云端都已变化，请手动选择",
		"status.download.detail":       "云端版本较新，可下载覆盖本地",
		"status.pending_upload.detail": "本地内容较新，可上传覆盖云端",

		// ── File list view ──────────────────────────────────────────────────
		"file_list.files_count":    "%d 个文件",
		"file_list.stats.matched":  "● 一致 %d",
		"file_list.stats.pending":  "● 待处理 %d",
		"file_list.stats.unbound":  "● 未关联 %d",
		"file_list.stats.matched_short": "一致 %d",
		"file_list.stats.pending_short": "待处理 %d",
		"file_list.stats.unbound_short": "未关联 %d",
		"file_list.page_indicator": "  第 %d/%d 页",
		"file_list.no_dir":         "(未关联目录)",
		"file_list.auto_sync_on":   "自动同步",
		"file_list.syncing":        "⟳ 同步中...",
		"file_list.last_sync":      "上次同步",

		// ── Empty state ─────────────────────────────────────────────────────
		"empty.no_storage": "  尚未配置存储服务",
		"empty.press":      "  按 ",
		"empty.configure":  " 配置 WebDAV 连接",
		"empty.no_files":   "  当前没有同步文件",
		"empty.add_hint":   " 添加文件到同步列表",
		"empty.drag_hint":  "  也可以直接将文件拖入窗口",

		// ── Bottom bar ──────────────────────────────────────────────────────
		"bottom.storage_unconfigured": "未配置",
		"bottom.syncing":              "⟳ 同步中",
		"bottom.auto_sync":            "auto %ds · %s",
		"bottom.navigate":             "↑↓ 导航",
		"bottom.sync":                 "Enter 同步",
		"bottom.upload":               "u 上传",
		"bottom.download":             "d 下载",
		"bottom.delete":               "x 删除",
		"bottom.dir":                  "e 编辑",
		"bottom.add":                  "a 添加",
		"bottom.settings":             "s 设置",
		"bottom.sync_all":             "y 全部",
		"bottom.auto":                 "o 自动",
		"bottom.refresh":              "r 刷新",
		"bottom.update":               "U 更新",
		"bottom.help":                 "? 帮助",
		"bottom.quit":                 "q 退出",

		// ── Add file view ───────────────────────────────────────────────────
		"add_file.title":              " ✚ 添加文件",
		"add_file.label.path":         "路径",
		"add_file.label.note":         "备注",
		"add_file.placeholder.path":   "文件路径（拖入或手动输入）",
		"add_file.placeholder.note":   "备注（可选）",
		"add_file.path_valid":         "✔ %s  %s",
		"add_file.path_invalid":       "✕ 路径不可达",
		"add_file.hint":               "  ↑↓/Tab 切换 · Enter 确认 · Ctrl+Y 复制 · Ctrl+U 清空 · Esc 返回 · 也可直接拖入文件",
		"add_file.error.invalid_path": "请输入有效的本地文件路径",
		"add_file.error.is_dir":       "不支持添加目录，请选择文件",
		"add_file.error.too_large":    "文件大于200MB，不支持同步",
		"add_file.error.duplicate":    "当前目录下同名文件已存在同步记录",
		"add_file.error.save_dir":     "保存本地目录失败: %s",
		"add_file.error.save":         "保存失败: %s",
		"add_file.added":              "已添加「%s」",
		"add_file.added_continue":     "✓ 已添加「%s」，继续添加下一个",

		// ── Edit file view ────────────────────────────────────────────────────
		"edit_file.title":             " ✏ 编辑文件",
		"edit_file.hint":              "  ↑↓/Tab 切换 · Enter 确认 · Ctrl+Y 复制 · Esc 返回",
		"edit_file.saved":             "「%s」编辑已保存",

		// ── Set directory view ────────────────────────────────────────────────
		"set_dir.title":           " 📂 设置本地目录",
		"set_dir.label":           "路径",
		"set_dir.placeholder":     "输入本地目录路径",
		"set_dir.hint":            "  Enter 确认 · Ctrl+Y 复制 · Ctrl+U 清空 · Esc 返回",
		"set_dir.error.empty":     "请输入目录路径",
		"set_dir.error.not_exist": "路径不存在: %s",
		"set_dir.error.not_dir":   "不是目录: %s",
		"set_dir.saved":           "目录已设置: %s",

		// ── Settings view ───────────────────────────────────────────────────
		"settings.title":                 " ⚙  存储设置",
		"settings.label.endpoint":        "WebDAV 地址",
		"settings.label.username":        "用户名",
		"settings.label.password":        "密码",
		"settings.label.base_path":       "远端目录",
		"settings.placeholder.endpoint":  "例如：https://dav.jianguoyun.com/dav",
		"settings.placeholder.username":  "坚果云用户名",
		"settings.placeholder.password":  "应用密码",
		"settings.placeholder.base_path": "留空使用默认 small-file-sync",
		"settings.base_path_default":     "留空默认 small-file-sync",
		"settings.password_show":         "[p 显示]",
		"settings.password_hide":         "[p 隐藏]",
		"settings.hint":                  "  ↑↓/Tab 切换 · Enter 保存 · t 测试 · p 密码 · Ctrl+B 导出 · Ctrl+Y 复制 · Ctrl+U 清空 · Esc 返回",
		"settings.hint_lang":             "  Tab 切换  Enter 切换语言  Esc 返回",
		"settings.error.required":        "WebDAV 需要填写地址、用户名和密码",
		"settings.test_failed":           "连接测试失败: %s",
		"settings.test_success":          "连接测试成功",
		"settings.saved":                 "存储设置已保存",
		"settings.language":              "语言",

		// ── Confirm dialog ──────────────────────────────────────────────────
		"confirm.cloud_overwrite.title":  "确认覆盖云端文件",
		"confirm.cloud_overwrite.msg":    "将使用本地版本覆盖云端文件内容，此操作不可撤销。",
		"confirm.cloud_overwrite.action": "覆盖云端",
		"confirm.local_overwrite.title":  "确认覆盖本地文件",
		"confirm.local_overwrite.msg":    "将使用云端版本覆盖本地文件内容，此操作不可撤销。",
		"confirm.local_overwrite.action": "覆盖本地",
		"confirm.delete.title":           "确认删除同步记录",
		"confirm.delete.msg":             "将删除「%s」的云端同步内容与记录，本地文件不会删除。",
		"confirm.delete.action":          "删除记录",

		// ── Sync actions ────────────────────────────────────────────────────
		"sync.action.upload":      "上传",
		"sync.action.download":    "下载",
		"sync.action.skip":        "跳过",
		"sync.action.unprocessed": "未处理",

		// ── Sync reasons ────────────────────────────────────────────────────
		"sync.reason.first_upload":        "首次上传成功",
		"sync.reason.cloud_not_available": "尚未首次上传，暂无可下载内容",
		"sync.reason.local_read_failed":   "读取本地文件失败",
		"sync.reason.local_restored":      "已恢复到本地",
		"sync.reason.already_synced":      "本地与云端一致",
		"sync.reason.both_missing":        "本地文件缺失，且云端无内容",
		"sync.reason.conflict_manual":     "本地与云端同时修改，请手动处理",
		"sync.reason.cloud_newer":         "云端较新",
		"sync.reason.overwrite_local":     "已覆盖本地文件",
		"sync.reason.local_newer":         "本地较新",
		"sync.reason.overwrite_cloud":     "已覆盖云端文件",
		"sync.reason.metadata_failed":     "保存元数据失败，已回滚",
		"sync.reason.upload_read_failed":  "上传后重新读取本地文件失败",

		// ── Sync result view ────────────────────────────────────────────────
		"sync.result.title":         " 📋 同步结果",
		"sync.result.auto":          "  (自动)",
		"sync.result.checked":       "检查 %d",
		"sync.result.uploaded":      "↑ 上传 %d",
		"sync.result.downloaded":    "↓ 下载 %d",
		"sync.result.skipped":       "– 跳过 %d",
		"sync.result.failed":        "✕ 失败 %d",
		"sync.result.unbound":       "○ 未关联 %d",
		"sync.result.header_file":   "文件",
		"sync.result.header_action": "操作",
		"sync.result.header_result": "结果",
		"sync.result.close_hint":    "  Enter / Esc  返回",
		"sync.result.syncing_hint": "  同步中，请稍候...",
		"sync.processing":          "处理中",
		"sync.already_synced":      "已同步，无需操作",

		// ── Refreshing / Auto sync ──────────────────────────────────────────
		"refreshing":         "正在刷新...",
		"auto_sync.enabled":  "已开启自动同步",
		"auto_sync.disabled": "已关闭自动同步",
		"auto_sync.done":     "自动同步完成",
		"auto_sync.no_change": "自动同步完成，无需变更",

		// ── Warn messages ───────────────────────────────────────────────────
		"warn.configure_storage":   "请先配置存储设置",
		"warn.configure_storage_s": "请先按 's' 配置存储设置",
		"warn.sync_in_progress":    "同步进行中，请等待完成",
		// ── Paste (drag-and-drop) ─────────────────────────────────────────────
		"paste.invalid_path": "路径不可达: %s",
		"paste.is_dir":       "不支持添加目录，请拖入文件",
		"paste.too_large":    "文件大于200MB，不支持同步",


		// ── Delete ──────────────────────────────────────────────────────────
		"delete.not_found": "未找到需要删除的记录",
		"delete.failed":    "删除失败: %s",
		"delete.success":   "已删除「%s」",

		// ── Error messages ──────────────────────────────────────────────────
		"error.load_file_list": "加载文件列表失败: %s",
		"error.save_local_dir": "保存本地目录失败: %s",
		"error.save_failed":    "保存失败: %s",

		// ── Help ────────────────────────────────────────────────────────────
		"help.title":              " ❓ 快捷键说明",
		"help.nav":                "导航",
		"help.nav.up":             "↑/k  上移",
		"help.nav.down":           "↓/j  下移",
		"help.nav.page_up":        "PgUp/←/h  上翻页",
		"help.nav.page_down":      "PgDn/→/l  下翻页",
		"help.nav.first_last":     "g/G  首/尾",
		"help.ops":                "操作",
		"help.ops.execute":        "Enter  执行",
		"help.ops.upload":         "u  上传",
		"help.ops.download":       "d  下载",
		"help.ops.delete":         "x  删除",
		"help.ops.note":           "n  备注",
		"help.ops.set_dir":        "e  编辑",
		"help.features":           "功能",
		"help.features.add":       "a  添加",
		"help.features.settings":  "s  设置",
		"help.features.sync_all":  "y  同步全部",
		"help.features.auto_sync": "o  自动同步",
		"help.features.refresh":   "r  刷新",
		"help.features.lang":      "L  Switch to English",
		"help.features.quit":      "q  退出",
		"help.features.update":    "U  检查更新",
		"help.general":            "通用",
		"help.general.copy":       "Ctrl+Y  复制路径/输入内容",
		"help.general.password":   "p  显示/隐藏密码",

		// ── Help hint ────────────────────────────────────────────────────────
		"help.hint_close":         "Esc / ?  返回",
		"confirm.hint_close":      "← → 选择  Enter 确认  Esc 返回",

		// ── Note view ────────────────────────────────────────────────────────
		"note.status":     "状态",
		"note.size":       "大小",
		"note.uploaded":   "上次上传",
		"note.user":       "上传者",
		"note.path":       "本地路径",
		"note.empty":      "（无备注）",
		"note.hint_close": "  Esc / n  返回",

		// ── WebDAV storage ──────────────────────────────────────────────────
		"webdav.endpoint_required": "请填写 WebDAV 地址",
		"webdav.username_required": "请填写 WebDAV 用户名",
		"webdav.verify_failed":     "连接成功，但读写校验失败",
		"webdav.connect_success":   "WebDAV 连接成功",
		"webdav.read_local_failed": "读取本地文件失败: %s",
		"webdav.file_too_large":    "文件大于200MB，不支持同步",
		"webdav.write_storage":     "写入共享存储失败: %s",
		"webdav.remote_empty":      "远端内容为空或分块缺失",
		"webdav.create_dir_failed": "创建本地目录失败: %s",
		"webdav.write_local":       "写回本地失败: %s",
		"webdav.readback_empty":    "远端读回为空",
		"webdav.readback_verify":   "远端读回校验失败",

		// ── uTools migration ────────────────────────────────────────────────
		"migrate.db_not_found":     "uTools 数据库目录未找到",
		"migrate.db_locked":        "无法打开 uTools 数据库（可能正在运行）: %v",
		"migrate.db_copy_failed":   "无法打开 uTools 数据库副本: %v",
		"migrate.plugin_not_found": "未在 uTools 数据库中找到 smallFileSync 插件数据",
		"migrate.uid_not_found":    "未找到 uid",
		"migrate.uid_format_error": "uid 格式异常",

		// ── Init ────────────────────────────────────────────────────────────
		"init.failed": "初始化失败: %v",
		"run.failed":  "运行失败: %v",

		// ── Export / Import ─────────────────────────────────────────────────
		"export.title":             " 📤 导出配置",
		"export.command_label":     "导入命令",
		"export.instruction":       "将此命令分享到其他机器运行即可导入当前配置（可手动选中上方命令复制）",
		"export.security_warning":  "⚠ 安全警告：此命令包含凭据信息（密码），请勿泄露！",
		"export.copied_hint":       "✓ 命令已复制到剪贴板",
		"export.temp_file":         "备用文件",
		"export.close_hint":        "  Esc / q  关闭",

		"import.prompt":            "导入此配置？(y/N): ",
		"import.cancelled":         "导入已取消。",
		"import.success":           "配置导入成功。",
		"import.security_warning":  "⚠ 警告：此配置包含敏感凭据，请勿泄露。",
		"import.field.language":    "语言",
		"import.field.auto_sync":   "自动同步",
		"import.field.storage_type":"存储类型",
		"import.field.endpoint":    "WebDAV 地址",
		"import.field.username":    "用户名",
		"import.field.password":    "密码",
		"import.field.base_path":   "远端目录",

		// ── Update ──────────────────────────────────────────────────────────
		"update.available":        "发现新版本 %s，当前版本 %s",
		"update.action":           "更新",
		"update.downloading":      "正在下载更新...",
		"update.success":          "更新成功",
		"update.restart_hint":     "新版本已就绪，请重启应用以使用新版本",
		"update.failed":           "更新失败: %s",
		"update.brew_hint":        "检测到通过 Homebrew 安装，请使用 brew upgrade sfs 更新",
		"update.check_failed":     "检查更新失败",
		"update.current_latest":   "已是最新版本 (%s)",
		"update.U":                "U 更新",
	},

	En: {
		// ── Common ──────────────────────────────────────────────────────────
		"common.success": "OK",
		"common.failure": "Failed",
		"common.error":   "Error",
		"common.warning": "Warning",
		"common.cancel":  "Cancel",
		"common.close":   "Close",

		// ── Toast ───────────────────────────────────────────────────────────
		"toast.copied":      "Copied to clipboard",
		"toast.copied_path": "Copied path: %s",
		"toast.no_dir":      "No local directory bound to this file",

		// ── File status keys ────────────────────────────────────────────────
		"status.matched":        "Synced",
		"status.pending_upload": "To Upload",
		"status.download":       "To Download",
		"status.initial_upload": "First Upload",
		"status.missing":        "Missing",
		"status.conflict":       "Conflict",
		"status.unbound":        "Unbound",

		// ── File status details ─────────────────────────────────────────────
		"status.unbound.detail":        "Set a local directory for this device first",
		"status.initial_upload.detail": "Perform the first upload",
		"status.missing.detail":        "Local file missing or unreadable, can restore from cloud",
		"status.matched.detail":        "Local and cloud are in sync",
		"status.conflict.detail":       "Both local and cloud changed, please choose manually",
		"status.download.detail":       "Cloud version is newer, can download to overwrite local",
		"status.pending_upload.detail": "Local version is newer, can upload to overwrite cloud",

		// ── File list view ──────────────────────────────────────────────────
		"file_list.files_count":    "%d files",
		"file_list.stats.matched":  "● Synced %d",
		"file_list.stats.pending":  "● Pending %d",
		"file_list.stats.unbound":  "● Unbound %d",
		"file_list.stats.matched_short": "Synced %d",
		"file_list.stats.pending_short": "Pending %d",
		"file_list.stats.unbound_short": "Unbound %d",
		"file_list.page_indicator": "  Page %d/%d",
		"file_list.no_dir":         "(No directory)",
		"file_list.auto_sync_on":   "Auto sync",
		"file_list.syncing":        "⟳ Syncing...",
		"file_list.last_sync":      "Last sync",

		// ── Empty state ─────────────────────────────────────────────────────
		"empty.no_storage": "  Storage not configured",
		"empty.press":      "  Press ",
		"empty.configure":  " to configure WebDAV",
		"empty.no_files":   "  No synced files yet",
		"empty.add_hint":   " to add files to sync list",
		"empty.drag_hint":  "  Or drag a file into the window",

		// ── Bottom bar ──────────────────────────────────────────────────────
		"bottom.storage_unconfigured": "Not configured",
		"bottom.syncing":              "⟳ Syncing",
		"bottom.auto_sync":            "auto %ds · %s",
		"bottom.navigate":             "↑↓ Nav",
		"bottom.sync":                 "Enter Sync",
		"bottom.upload":               "u Upload",
		"bottom.download":             "d Download",
		"bottom.delete":               "x Delete",
		"bottom.dir":                  "e Edit",
		"bottom.add":                  "a Add",
		"bottom.settings":             "s Settings",
		"bottom.sync_all":             "y All",
		"bottom.auto":                 "o Auto",
		"bottom.refresh":              "r Refresh",
		"bottom.update":               "U Update",
		"bottom.help":                 "? Help",
		"bottom.quit":                 "q Quit",

		// ── Add file view ───────────────────────────────────────────────────
		"add_file.title":              " ✚ Add File",
		"add_file.label.path":         "Path",
		"add_file.label.note":         "Note",
		"add_file.placeholder.path":   "File path (drag in or type manually)",
		"add_file.placeholder.note":   "Note (optional)",
		"add_file.path_valid":         "✔ %s  %s",
		"add_file.path_invalid":       "✕ Path not accessible",
		"add_file.hint":               "  ↑↓/Tab Switch · Enter Confirm · Ctrl+Y Copy · Ctrl+U Clear · Esc Back · Or drag a file in",
		"add_file.error.invalid_path": "Please enter a valid local file path",
		"add_file.error.is_dir":       "Directories not supported, please select a file",
		"add_file.error.too_large":    "File exceeds 200MB, sync not supported",
		"add_file.error.duplicate":    "A file with the same name already exists in this directory",
		"add_file.error.save_dir":     "Failed to save local directory: %s",
		"add_file.error.save":         "Save failed: %s",
		"add_file.added":              "Added \"%s\"",
		"add_file.added_continue":     "✓ Added \"%s\", continue adding next",

		// ── Edit file view ────────────────────────────────────────────────────
		"edit_file.title":             " ✏ Edit File",
		"edit_file.hint":              "  ↑↓/Tab Switch · Enter Confirm · Ctrl+Y Copy · Esc Back",
		"edit_file.saved":             "\"%s\" saved",

		// ── Set directory view ────────────────────────────────────────────────
		"set_dir.title":           " 📂 Set Local Directory",
		"set_dir.label":           "Path",
		"set_dir.placeholder":     "Enter local directory path",
		"set_dir.hint":            "  Enter Confirm · Ctrl+Y Copy · Ctrl+U Clear · Esc Back",
		"set_dir.error.empty":     "Please enter a directory path",
		"set_dir.error.not_exist": "Path does not exist: %s",
		"set_dir.error.not_dir":   "Not a directory: %s",
		"set_dir.saved":           "Directory set: %s",

		// ── Settings view ───────────────────────────────────────────────────
		"settings.title":                 " ⚙  Storage Settings",
		"settings.label.endpoint":        "WebDAV URL",
		"settings.label.username":        "Username",
		"settings.label.password":        "Password",
		"settings.label.base_path":       "Remote Path",
		"settings.placeholder.endpoint":  "e.g. https://dav.jianguoyun.com/dav",
		"settings.placeholder.username":  "Jianguoyun username",
		"settings.placeholder.password":  "App password",
		"settings.placeholder.base_path": "Leave empty for default small-file-sync",
		"settings.base_path_default":     "Default: small-file-sync",
		"settings.password_show":         "[p Show]",
		"settings.password_hide":         "[p Hide]",
		"settings.hint":                  "  ↑↓/Tab Switch · Enter Save · t Test · p Password · Ctrl+B Export · Ctrl+Y Copy · Ctrl+U Clear · Esc Back",
		"settings.hint_lang":             "  Tab Switch  Enter Toggle Language  Esc Back",
		"settings.error.required":        "WebDAV requires URL, username, and password",
		"settings.test_failed":           "Connection test failed: %s",
		"settings.test_success":          "Connection successful",
		"settings.saved":                 "Storage settings saved",
		"settings.language":              "Language",

		// ── Confirm dialog ──────────────────────────────────────────────────
		"confirm.cloud_overwrite.title":  "Confirm Overwrite Cloud",
		"confirm.cloud_overwrite.msg":    "This will overwrite the cloud version with the local version. This cannot be undone.",
		"confirm.cloud_overwrite.action": "Overwrite Cloud",
		"confirm.local_overwrite.title":  "Confirm Overwrite Local",
		"confirm.local_overwrite.msg":    "This will overwrite the local version with the cloud version. This cannot be undone.",
		"confirm.local_overwrite.action": "Overwrite Local",
		"confirm.delete.title":           "Confirm Delete Record",
		"confirm.delete.msg":             "This will delete cloud sync content and record for \"%s\". Local files will not be deleted.",
		"confirm.delete.action":          "Delete Record",

		// ── Sync actions ────────────────────────────────────────────────────
		"sync.action.upload":      "Upload",
		"sync.action.download":    "Download",
		"sync.action.skip":        "Skip",
		"sync.action.unprocessed": "Unprocessed",

		// ── Sync reasons ────────────────────────────────────────────────────
		"sync.reason.first_upload":        "First upload successful",
		"sync.reason.cloud_not_available": "Not yet uploaded, nothing to download",
		"sync.reason.local_read_failed":   "Failed to read local file",
		"sync.reason.local_restored":      "Restored to local",
		"sync.reason.already_synced":      "Local and cloud are in sync",
		"sync.reason.both_missing":        "Local file missing and no cloud content",
		"sync.reason.conflict_manual":     "Both modified, please resolve manually",
		"sync.reason.cloud_newer":         "Cloud version is newer",
		"sync.reason.overwrite_local":     "Overwrote local file",
		"sync.reason.local_newer":         "Local version is newer",
		"sync.reason.overwrite_cloud":     "Overwrote cloud file",
		"sync.reason.metadata_failed":     "Failed to save metadata, rolled back",
		"sync.reason.upload_read_failed":  "Failed to re-read local file after upload",

		// ── Sync result view ────────────────────────────────────────────────
		"sync.result.title":         " 📋 Sync Result",
		"sync.result.auto":          "  (Auto)",
		"sync.result.checked":       "Checked %d",
		"sync.result.uploaded":      "↑ Uploaded %d",
		"sync.result.downloaded":    "↓ Downloaded %d",
		"sync.result.skipped":       "– Skipped %d",
		"sync.result.failed":        "✕ Failed %d",
		"sync.result.unbound":       "○ Unbound %d",
		"sync.result.header_file":   "File",
		"sync.result.header_action": "Action",
		"sync.result.header_result": "Result",
		"sync.result.close_hint":    "  Enter / Esc  Back",
		"sync.result.syncing_hint": "  Syncing, please wait...",
		"sync.processing":          "Processing",
		"sync.already_synced":      "Already in sync",

		// ── Refreshing / Auto sync ──────────────────────────────────────────
		"refreshing":         "Refreshing...",
		"auto_sync.enabled":  "Auto sync enabled",
		"auto_sync.disabled": "Auto sync disabled",
		"auto_sync.done":     "Auto sync done",
		"auto_sync.no_change": "Auto sync done, no changes",

		// ── Warn messages ───────────────────────────────────────────────────
		"warn.configure_storage":   "Please configure storage settings first",
		"warn.configure_storage_s": "Please press 's' to configure storage settings",
		"warn.sync_in_progress":    "Sync in progress, please wait",
		// ── Paste (drag-and-drop) ─────────────────────────────────────────────
		"paste.invalid_path": "Path not accessible: %s",
		"paste.is_dir":       "Directories not supported, please drag a file",
		"paste.too_large":    "File exceeds 200MB, sync not supported",


		// ── Delete ──────────────────────────────────────────────────────────
		"delete.not_found": "Record not found",
		"delete.failed":    "Delete failed: %s",
		"delete.success":   "Deleted \"%s\"",

		// ── Error messages ──────────────────────────────────────────────────
		"error.load_file_list": "Failed to load file list: %s",
		"error.save_local_dir": "Failed to save local directory: %s",
		"error.save_failed":    "Save failed: %s",

		// ── Help ────────────────────────────────────────────────────────────
		"help.title":              " ❓ Keyboard Shortcuts",
		"help.nav":                "Navigation",
		"help.nav.up":             "↑/k  Up",
		"help.nav.down":           "↓/j  Down",
		"help.nav.page_up":        "PgUp/←/h  Page Up",
		"help.nav.page_down":      "PgDn/→/l  Page Down",
		"help.nav.first_last":     "g/G  First/Last",
		"help.ops":                "Operations",
		"help.ops.execute":        "Enter  Execute",
		"help.ops.upload":         "u  Upload",
		"help.ops.download":       "d  Download",
		"help.ops.delete":         "x  Delete",
		"help.ops.note":           "n  Note",
		"help.ops.set_dir":        "e  Edit",
		"help.features":           "Features",
		"help.features.add":       "a  Add",
		"help.features.settings":  "s  Settings",
		"help.features.sync_all":  "y  Sync All",
		"help.features.auto_sync": "o  Auto Sync",
		"help.features.refresh":   "r  Refresh",
		"help.features.lang":      "L  Switch to 中文",
		"help.features.quit":      "q  Quit",
		"help.features.update":    "U  Check for updates",
		"help.general":            "General",
		"help.general.copy":       "Ctrl+Y  Copy path / input",
		"help.general.password":   "p  Show/Hide password",

		// ── Help hint ────────────────────────────────────────────────────────
		"help.hint_close":         "Esc / ?  Back",
		"confirm.hint_close":      "← → Select  Enter Confirm  Esc Back",

		// ── Note view ────────────────────────────────────────────────────────
		"note.status":     "Status",
		"note.size":       "Size",
		"note.uploaded":   "Last Upload",
		"note.user":       "User",
		"note.path":       "Local Path",
		"note.empty":      "(No note)",
		"note.hint_close": "  Esc / n  Back",

		// ── WebDAV storage ──────────────────────────────────────────────────
		"webdav.endpoint_required": "Please enter WebDAV URL",
		"webdav.username_required": "Please enter WebDAV username",
		"webdav.verify_failed":     "Connected but read/write verification failed",
		"webdav.connect_success":   "WebDAV connection successful",
		"webdav.read_local_failed": "Failed to read local file: %s",
		"webdav.file_too_large":    "File exceeds 200MB, sync not supported",
		"webdav.write_storage":     "Failed to write to shared storage: %s",
		"webdav.remote_empty":      "Remote content is empty or chunks are missing",
		"webdav.create_dir_failed": "Failed to create local directory: %s",
		"webdav.write_local":       "Failed to write back to local: %s",
		"webdav.readback_empty":    "Remote readback is empty",
		"webdav.readback_verify":   "Remote readback verification failed",

		// ── uTools migration ────────────────────────────────────────────────
		"migrate.db_not_found":     "uTools database directory not found",
		"migrate.db_locked":        "Cannot open uTools database (may be running): %v",
		"migrate.db_copy_failed":   "Cannot open uTools database copy: %v",
		"migrate.plugin_not_found": "smallFileSync plugin data not found in uTools database",
		"migrate.uid_not_found":    "UID not found",
		"migrate.uid_format_error": "UID format error",

		// ── Init ────────────────────────────────────────────────────────────
		"init.failed": "Initialization failed: %v",
		"run.failed":  "Runtime error: %v",

		// ── Export / Import ─────────────────────────────────────────────────
		"export.title":             " 📤 Export Config",
		"export.command_label":     "Import Command",
		"export.instruction":       "Share this command and run it on another machine to import the config (select the command above to copy manually)",
		"export.security_warning":  "⚠ Security Warning: This command contains credentials (password), do not share it!",
		"export.copied_hint":       "✓ Command copied to clipboard",
		"export.temp_file":         "Fallback file",
		"export.close_hint":        "  Esc / q  Close",

		"import.prompt":            "Import this configuration? (y/N): ",
		"import.cancelled":         "Import cancelled.",
		"import.success":           "Configuration imported successfully.",
		"import.security_warning":  "⚠ Warning: This configuration contains sensitive credentials. Do not share it.",
		"import.field.language":    "Language",
		"import.field.auto_sync":   "Auto Sync",
		"import.field.storage_type":"Storage Type",
		"import.field.endpoint":    "WebDAV Endpoint",
		"import.field.username":    "Username",
		"import.field.password":    "Password",
		"import.field.base_path":   "Remote Path",

		// ── Update ──────────────────────────────────────────────────────────
		"update.available":        "New version %s available (current: %s)",
		"update.action":           "Update",
		"update.downloading":      "Downloading update...",
		"update.success":          "Update successful",
		"update.restart_hint":     "New version is ready, please restart the app to use it",
		"update.failed":           "Update failed: %s",
		"update.brew_hint":        "Homebrew install detected, use brew upgrade sfs to update",
		"update.check_failed":     "Failed to check for updates",
		"update.current_latest":   "Already on the latest version (%s)",
		"update.U":                "U Update",
	},
}
