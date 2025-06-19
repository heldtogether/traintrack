package uploads

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	CreateQuery = `INSERT INTO uploads (files) VALUES ($1) RETURNING id`
	UpdateQuery = `UPDATE uploads SET files = $1 WHERE id = $2`
	GetQuery    = `SELECT id, files FROM uploads WHERE id = $1`
)

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Repository struct {
	conn Querier
}

func NewRepository(conn Querier) *Repository {
	return &Repository{
		conn: conn,
	}
}

func (r *Repository) Create(u *Upload) (*Upload, error) {
	filesJSON, err := json.Marshal(u.Files)
	if err != nil {
		// I'm not sure how this marhsalling would actually fail
		return nil, err
	}

	query := CreateQuery
	row := r.conn.QueryRow(
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

func (r *Repository) Move(u *Upload) error {
	return r.MoveWithQuerier(r.conn, u)
}

func (r *Repository) MoveWithQuerier(conn Querier, u *Upload) error {
	filesJSON, err := json.Marshal(u.Files)
	if err != nil {
		// Shouldn't really happen, but good to check
		return err
	}

	_, err = conn.Exec(
		context.Background(),
		UpdateQuery,
		filesJSON,
		u.ID,
	)

	return err
}

func (r *Repository) GetByIDWithQuerier(q Querier, id string) (*Upload, error) {
	row := q.QueryRow(context.Background(), GetQuery, id)

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
