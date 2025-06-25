package uploads

import (
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

/*
FileSystemStore is a local file system storage provider. It implements ReadSaver.
*/
type FileSystemStore struct {
	BaseDir string
}

func (f *FileSystemStore) SaveFile(dstPath string, file multipart.File) error {
	fullPath := filepath.Join(f.BaseDir, dstPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	dst, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	return err
}

func (f *FileSystemStore) MoveFile(srcPath string, dstPath string) error {
	fullSrcPath := filepath.Join(f.BaseDir, srcPath)
	fullDstPath := filepath.Join(f.BaseDir, dstPath)

	if err := os.MkdirAll(filepath.Dir(fullDstPath), 0755); err != nil {
		return err
	}

	return os.Rename(fullSrcPath, fullDstPath)
}

func (f *FileSystemStore) ReadFile(path string) ([]byte, error) {
	fullPath := filepath.Join(f.BaseDir, path)
	return os.ReadFile(fullPath)
}
