package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomic_Success(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "state.json")
	data := []byte("hello")

	if err := WriteAtomic(target, data, 0o644); err != nil {
		t.Fatalf("WriteAtomic returned error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("failed reading target: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("content mismatch: got %q, want %q", got, data)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("permissions = %v, want %v", info.Mode().Perm(), os.FileMode(0o644))
	}

	tmpFiles, err := filepath.Glob(filepath.Join(dir, ".tmp-*"))
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}
	if len(tmpFiles) != 0 {
		t.Fatalf("expected temp files cleaned up, found %v", tmpFiles)
	}
}

func TestWriteAtomic_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "state.json")

	if err := WriteAtomic(target, []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteAtomic returned error: %v", err)
	}

	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target missing: %v", err)
	}

	info, err := os.Stat(filepath.Dir(target))
	if err != nil {
		t.Fatalf("stat dir failed: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("dir perm = %v, want %v", info.Mode().Perm(), os.FileMode(0o755))
	}
}

func TestWriteAtomic_CleanupAndPreserveOnError(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "state.json")
	if err := os.WriteFile(target, []byte("original"), 0o644); err != nil {
		t.Fatalf("seed state file: %v", err)
	}

	// Make directory read-only to force failure during temp file creation.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}

	err := WriteAtomic(target, []byte("new"), 0o644)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Restore permissions to inspect contents.
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("restore chmod: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	if string(data) != "original" {
		t.Fatalf("original file modified on error: %q", data)
	}

	tmpFiles, err := filepath.Glob(filepath.Join(dir, ".tmp-*"))
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}
	if len(tmpFiles) != 0 {
		t.Fatalf("expected temp cleanup on error, found %v", tmpFiles)
	}
}
