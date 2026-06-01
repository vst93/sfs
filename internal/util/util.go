package util

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
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

// SplitIntoChunks splits data into base64-encoded chunks of ~700KB.
func SplitIntoChunks(data []byte, chunkSize int) [][]byte {
	if chunkSize <= 0 {
		chunkSize = 700 * 1024
	}
	var chunks [][]byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	if len(chunks) == 0 {
		chunks = append(chunks, data)
	}
	return chunks
}
