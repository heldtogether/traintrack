package models

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/heldtogether/traintrack/internal/uploads"
	"github.com/jackc/pgx/v5"
)

type modelsStore interface {
	createWithQuerier(q Querier, m *Model) (*Model, error)
	List() ([]*Model, error)
}

/*
An interface that allows an Upload to be moved using the provider Querier
which may be a transaction.
*/
type UploadMover interface {
	GetByIDWithQuerier(q uploads.Querier, id string) (*uploads.Upload, error)
	MoveWithQuerier(q uploads.Querier, u *uploads.Upload) error
}

type FileMover interface {
	MoveFile(srcPath, dstPath string) error
}

type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type DefaultCreator struct {
	s           modelsStore
	uploadMover UploadMover
	fileMover   FileMover
	db          TxBeginner
}

func NewCreator(s *Store, u UploadMover, f FileMover, db TxBeginner) *DefaultCreator {
	return &DefaultCreator{
		s:           s,
		uploadMover: u,
		fileMover:   f,
		db:          db,
	}
}

/*
Create a new model and move any artefacts from temporary storage
to a sensible forever home.
*/
func (c *DefaultCreator) Create(ctx context.Context, m *Model) (created *Model, err error) {
	tx, err := c.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	created, err = c.s.createWithQuerier(tx, m)
	if err != nil {
		return nil, err
	}

	for _, id := range m.UploadIds {
		upload, err := c.uploadMover.GetByIDWithQuerier(tx, id)
		if err != nil {
			return nil, fmt.Errorf("get upload %s: %w", id, err)
		}

		newFiles := make(map[string]uploads.FileRef, len(upload.Files))
		for name, file := range upload.Files {
			origPath := filepath.Join(file.Path, file.FileName)
			newPath := filepath.Join("models", created.ID)
			if err := c.fileMover.MoveFile(origPath, filepath.Join(newPath, file.FileName)); err != nil {
				return nil, fmt.Errorf("move file %s: %w", file, err)
			}
			newFiles[name] = uploads.FileRef{
				Provider: file.Provider,
				FileName: file.FileName,
				Path:     newPath,
			}
		}

		upload.Files = newFiles
		upload.ModelID = pointerTo(created.ID)
		if err := c.uploadMover.MoveWithQuerier(tx, upload); err != nil {
			return nil, fmt.Errorf("update upload %s: %w", id, err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return created, nil
}

func pointerTo[T any](v T) *T {
	return &v
}
