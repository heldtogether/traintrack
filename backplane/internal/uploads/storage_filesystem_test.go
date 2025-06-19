package uploads

import (
	"bytes"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
)

type mockMultipartFile struct {
	*bytes.Reader
}

func (m *mockMultipartFile) Close() error {
	return nil
}

// Ensure it implements multipart.File
var _ multipart.File = (*mockMultipartFile)(nil)

func newMockMultipartFile(content []byte) multipart.File {
	return &mockMultipartFile{Reader: bytes.NewReader(content)}
}

func TestFileSystemStorage_SaveFile(t *testing.T) {
	tmpDir := t.TempDir()

	storage := &FileSystemStorage{BaseDir: tmpDir}

	content := []byte("hello world")
	file := newMockMultipartFile(content)

	dstPath := "some/nested/file.txt"

	err := storage.SaveFile(dstPath, file)
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	fullPath := filepath.Join(tmpDir, dstPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if string(data) != string(content) {
		t.Errorf("file content mismatch: got %q, want %q", string(data), string(content))
	}
}

func TestFileSystemStorage_SaveFile_DirCreationFails(t *testing.T) {
	// Most systems disallow writing to /dev/null/file.txt
	storage := &FileSystemStorage{BaseDir: "/dev/null"}

	file := newMockMultipartFile([]byte("test content"))
	err := storage.SaveFile("some/path/file.txt", file)
	if err == nil {
		t.Fatal("expected error due to unwritable directory, got nil")
	}
}

func TestFileSystemStorage_SaveFile_FileCreationFails(t *testing.T) {
	// Create a temp dir with read-only permissions
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(nestedDir, 0500); err != nil {
		t.Fatalf("failed to create readonly dir: %v", err)
	}

	storage := &FileSystemStorage{BaseDir: tmpDir}
	file := newMockMultipartFile([]byte("test"))

	dstPath := filepath.Join("readonly", "file.txt")
	err := storage.SaveFile(dstPath, file)
	if err == nil {
		t.Fatal("expected file creation error due to read-only directory, got nil")
	}
}

func TestFileSystemStorage_MoveFile(t *testing.T) {
	tmpDir := t.TempDir()

	storage := &FileSystemStorage{BaseDir: tmpDir}

	srcPath := "original/location/file.txt"
	dstPath := "new/location/file.txt"
	content := []byte("move me")

	err := storage.SaveFile(srcPath, newMockMultipartFile(content))
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	err = storage.MoveFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("MoveFile failed: %v", err)
	}

	fullDstPath := filepath.Join(tmpDir, dstPath)
	data, err := os.ReadFile(fullDstPath)
	if err != nil {
		t.Fatalf("failed to read moved file: %v", err)
	}

	if string(data) != string(content) {
		t.Errorf("file content mismatch after move: got %q, want %q", string(data), string(content))
	}

	fullSrcPath := filepath.Join(tmpDir, srcPath)
	if _, err := os.Stat(fullSrcPath); !os.IsNotExist(err) {
		t.Errorf("source file still exists after move: %v", err)
	}
}
