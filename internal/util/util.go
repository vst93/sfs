package util

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"
)

// GenerateUID creates a new unique identifier.
func GenerateUID() string {
	return uuid.New().String()
}

// CalculateFileMD5 computes the MD5 hash of a file.
func CalculateFileMD5(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// FileSeparator returns the OS path separator.
func FileSeparator() string {
	if runtime.GOOS == "windows" {
		return "\\"
	}
	return "/"
}

// ExpandUserPath expands a leading ~/ (or ~\ on Windows) using the current
// user's home directory. Other tilde forms such as ~other-user are unchanged.
func ExpandUserPath(path string) string {
	path = strings.TrimSpace(path)
	if path != "~" && (len(path) < 2 || path[0] != '~' || !os.IsPathSeparator(path[1])) {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}

// NormalizeLocalPath expands the home directory and resolves relative paths
// against the current working directory before storing them.
func NormalizeLocalPath(path string) (string, error) {
	path = ExpandUserPath(path)
	if path == "" {
		return "", nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

// CurrentUsername returns the current OS username.
func CurrentUsername() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	if u := os.Getenv("USERNAME"); u != "" {
		return u
	}
	return "unknown"
}

// NormalizeStorageBasePath normalizes a WebDAV base path.
func NormalizeStorageBasePath(basePath string) string {
	raw := strings.TrimSpace(basePath)
	if raw == "" {
		return "small-file-sync"
	}
	if raw == "/" || raw == "." {
		return ""
	}
	parts := strings.Split(raw, "/")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return strings.Join(result, "/")
}

// CalculateFileMD5FromBytes computes the MD5 hash of a byte slice.
func CalculateFileMD5FromBytes(data []byte) (string, error) {
	h := md5.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil)), nil
}
