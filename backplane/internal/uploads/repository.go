package uploads

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

const (
	CreateQuery = `INSERT INTO uploads (files) VALUES ($1) RETURNING id`
)

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
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
