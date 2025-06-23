package datasets

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/heldtogether/traintrack/internal/uploads"
	"github.com/jackc/pgx/v5"
)

type DatasetRepo interface {
	CreateWithQuerier(q Querier, d *Dataset) (*Dataset, error)
	List() ([]*Dataset, error)
}

type UploadRepo interface {
	GetByIDWithQuerier(q uploads.Querier, id string) (*uploads.Upload, error)
	MoveWithQuerier(q uploads.Querier, u *uploads.Upload) error
}

type Storage interface {
	MoveFile(srcPath, dstPath string) error
}

type DB interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Tx interface {
	Querier
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type Service struct {
	DatasetsRepo DatasetRepo
	UploadsRepo  UploadRepo
	Storage      Storage
	DB           DB
}

// Create a new dataset and move any artefacts from temporary storage
// to a sensible forever home.
func (s *Service) Create(ctx context.Context, d *Dataset) (created *Dataset, err error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	created, err = s.DatasetsRepo.CreateWithQuerier(tx, d)
	if err != nil {
		return nil, err
	}

	for _, id := range d.UploadIds {
		upload, err := s.UploadsRepo.GetByIDWithQuerier(tx, id)
		if err != nil {
			return nil, fmt.Errorf("get upload %s: %w", id, err)
		}

		newFiles := make(map[string]uploads.FileRef, len(upload.Files))
		for name, file := range upload.Files {
			origPath := filepath.Join(file.Path, file.FileName)
			newPath := filepath.Join("datasets", created.ID)
			if err := s.Storage.MoveFile(origPath, filepath.Join(newPath, file.FileName)); err != nil {
				return nil, fmt.Errorf("move file %s: %w", file, err)
			}
			newFiles[name] = uploads.FileRef{
				Provider: file.Provider,
				FileName: file.FileName,
				Path:     newPath,
			}
		}

		upload.Files = newFiles
		upload.DatasetID = created.ID
		if err := s.UploadsRepo.MoveWithQuerier(tx, upload); err != nil {
			return nil, fmt.Errorf("update upload %s: %w", id, err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) List() ([]*Dataset, error) {
	return s.DatasetsRepo.List()
}
