package uploads

import (
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

type FileSystemStorage struct {
	BaseDir string
}

func (f *FileSystemStorage) SaveFile(dstPath string, file multipart.File) error {
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

func (f *FileSystemStorage) MoveFile(srcPath string, dstPath string) error {
	fullSrcPath := filepath.Join(f.BaseDir, srcPath)
	fullDstPath := filepath.Join(f.BaseDir, dstPath)

	if err := os.MkdirAll(filepath.Dir(fullDstPath), 0755); err != nil {
		return err
	}

	return os.Rename(fullSrcPath, fullDstPath)
}
