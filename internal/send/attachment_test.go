package send

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadLocalAttachment(t *testing.T) {
	t.Parallel()

	t.Run("empty path", func(t *testing.T) {
		t.Parallel()
		if _, err := LoadLocalAttachment(""); err == nil || !strings.Contains(err.Error(), "cannot be empty") {
			t.Fatalf("LoadLocalAttachment(\"\") expected empty-path error, got %v", err)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		t.Parallel()
		_, err := LoadLocalAttachment(filepath.Join(t.TempDir(), "missing.txt"))
		if err == nil || !strings.Contains(err.Error(), "stat attachment") {
			t.Fatalf("LoadLocalAttachment(missing) expected stat error, got %v", err)
		}
	})

	t.Run("file is a directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		_, err := LoadLocalAttachment(dir)
		if err == nil || !strings.Contains(err.Error(), "is a directory") {
			t.Fatalf("LoadLocalAttachment(dir) expected directory error, got %v", err)
		}
	})

	t.Run("content with known extension", func(t *testing.T) {
		t.Parallel()
		content := []byte("hello attachment")
		path := filepath.Join(t.TempDir(), "note.txt")
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}

		att, err := LoadLocalAttachment(path)
		if err != nil {
			t.Fatalf("LoadLocalAttachment() unexpected error: %v", err)
		}
		if att.Filename != "note.txt" {
			t.Errorf("Filename = %q, want %q", att.Filename, "note.txt")
		}
		if att.ContentType != "text/plain; charset=utf-8" {
			t.Errorf("ContentType = %q, want %q", att.ContentType, "text/plain; charset=utf-8")
		}
		if want := base64.StdEncoding.EncodeToString(content); att.Content != want {
			t.Errorf("Content = %q, want %q", att.Content, want)
		}
	})

	t.Run("unknown extension falls back to content sniffing", func(t *testing.T) {
		t.Parallel()
		content := []byte("plain text without extension")
		path := filepath.Join(t.TempDir(), "data.unknownext")
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}

		att, err := LoadLocalAttachment(path)
		if err != nil {
			t.Fatalf("LoadLocalAttachment() unexpected error: %v", err)
		}
		if !strings.HasPrefix(att.ContentType, "text/plain") {
			t.Errorf("ContentType = %q, want text/plain prefix (sniffed)", att.ContentType)
		}
	})
}