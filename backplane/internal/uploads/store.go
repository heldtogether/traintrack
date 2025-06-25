package uploads

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Upload struct {
	ID        string             `json:"id"`
	Files     map[string]FileRef `json:"files"`
	DatasetID *string            `json:"dataset_id,omitempty"`
	ModelID   *string            `json:"model_id,omitempty"`
}

/*
Provider allows configuration of the underlying file storage provider.
*/
type Provider string

const (
	ProviderUnknown    Provider = "unknown"
	ProviderFileSystem Provider = "filesystem"
)

type FileRef struct {
	Provider Provider `json:"provider"`
	FileName string   `json:"filename"`
	Path     string   `json:"path"`
}

const (
	createQuery = `INSERT INTO uploads (files) VALUES ($1) RETURNING id`
	updateQuery = `UPDATE uploads SET files = $1, dataset_id = $2, model_id = $3 WHERE id = $4`
	getQuery    = `SELECT id, files FROM uploads WHERE id = $1`
)

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Store struct {
	q Querier
}

func NewStore(q Querier) *Store {
	return &Store{
		q: q,
	}
}

func (s *Store) Create(u *Upload) (*Upload, error) {
	filesJSON, err := json.Marshal(u.Files)
	if err != nil {
		// I'm not sure how this marhsalling would actually fail
		return nil, err
	}

	query := createQuery
	row := s.q.QueryRow(
		context.Background(),
		query,
		filesJSON,
	)

	var id string
	if err := row.Scan(&id); err != nil {
		return nil, err
	}

	return &Upload{
		ID:    id,
		Files: u.Files,
	}, nil
}

func (s *Store) Get(id string) (*Upload, error) {
	query := getQuery
	row := s.q.QueryRow(
		context.Background(),
		query,
		id,
	)

	var upload Upload
	if err := row.Scan(&upload.ID, &upload.Files); err != nil {
		return nil, err
	}

	return &upload, nil
}

func (s *Store) Move(u *Upload) error {
	return s.MoveWithQuerier(s.q, u)
}

func (s *Store) MoveWithQuerier(conn Querier, u *Upload) error {
	filesJSON, err := json.Marshal(u.Files)
	if err != nil {
		// Shouldn't really happen, but good to check
		return err
	}

	_, err = conn.Exec(
		context.Background(),
		updateQuery,
		filesJSON,
		u.DatasetID,
		u.ModelID,
		u.ID,
	)

	return err
}

func (s *Store) GetByIDWithQuerier(q Querier, id string) (*Upload, error) {
	row := q.QueryRow(context.Background(), getQuery, id)

	var upload Upload
	var filesJSON []byte

	if err := row.Scan(&upload.ID, &filesJSON); err != nil {
		return nil, fmt.Errorf("scan upload: %w", err)
	}

	if err := json.Unmarshal(filesJSON, &upload.Files); err != nil {
		return nil, fmt.Errorf("unmarshal files: %w", err)
	}

	return &upload, nil
}
