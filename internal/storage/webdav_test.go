package storage

import (
	"strings"
	"testing"
)

func TestVersionedFileDataStorageKeysAreUnique(t *testing.T) {
	first := buildVersionedFileDataStorageKey("record-id")
	second := buildVersionedFileDataStorageKey("record-id")
	if first == second {
		t.Fatalf("uploads reused key %q", first)
	}
	if !strings.HasPrefix(first, "file_record-id_") || !strings.HasPrefix(second, "file_record-id_") {
		t.Fatalf("unexpected versioned keys: %q, %q", first, second)
	}
}
