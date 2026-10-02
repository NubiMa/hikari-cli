package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMoveOrCopy(t *testing.T) {
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "source_bin")
	dst := filepath.Join(tmpDir, "dest_bin")

	content := []byte("#!/bin/sh\necho hello\n")
	if err := os.WriteFile(src, content, 0755); err != nil {
		t.Fatalf("writing source file: %v", err)
	}

	if err := moveOrCopy(src, dst); err != nil {
		t.Fatalf("moveOrCopy failed: %v", err)
	}

	// Verify src is gone
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("expected source to be removed, got: %v", err)
	}

	// Verify dst exists and has content
	read, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading dest file: %v", err)
	}
	if string(read) != string(content) {
		t.Errorf("dest content mismatch: got %q, want %q", string(read), string(content))
	}

	// Verify executable permissions.
	// Windows does not support Unix execute bits; os.Chmod only controls the
	// read-only attribute there, so this check is skipped on Windows.
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(dst)
		if err != nil {
			t.Fatalf("stat dest file: %v", err)
		}
		if fi.Mode().Perm()&0111 == 0 {
			t.Errorf("expected executable permissions, got: %v", fi.Mode().Perm())
		}
	}
}
