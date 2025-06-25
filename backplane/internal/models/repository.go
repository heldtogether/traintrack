package models

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const (
	CreateQuery = `INSERT INTO 
models (name, parent, version, description, dataset, config, metadata, environment, evaluation) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	ListQuery   = `SELECT 
  m.id,
  m.name,
  m.parent,
  m.version,
  m.description,
	COALESCE(
    jsonb_object_agg(file_key, u.id) FILTER (WHERE file_key IS NOT NULL),
    '{}'::jsonb
  ) AS artefacts
FROM models m
LEFT JOIN uploads u ON u.model_id = m.id
LEFT JOIN LATERAL jsonb_object_keys(u.files) AS file_key ON true
GROUP BY m.id, m.name, m.parent, m.version, m.description;`
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

func (r *Repository) Create(d *Model) (*Model, error) {
	return r.CreateWithQuerier(r.conn, d)
}

func (r *Repository) CreateWithQuerier(conn Querier, m *Model) (*Model, error) {
	query := CreateQuery
	row := conn.QueryRow(
		context.Background(),
		query,
		m.Name,
		m.Parent,
		m.Version,
		m.Description,
		m.DatasetId,
		m.Config,
		m.Metadata,
		m.Environment,
		m.Evaluation,
	)

	var id string
	if err := row.Scan(&id); err != nil {
		return nil, err
	}

	return &Model{
		ID:          id,
		Name:        m.Name,
		Parent:      m.Parent,
		Version:     m.Version,
		Description: m.Description,
	}, nil
}

func (r *Repository) List() ([]*Model, error) {
	rows, err := r.conn.Query(
		context.TODO(),
		ListQuery,
	)
	if err != nil {
		return nil, fmt.Errorf("could not query models: %s", err)
	}

	defer rows.Close()

	ms := []*Model{}
	for rows.Next() {
		m := &Model{}
		if err := rows.Scan(
			&m.ID,
			&m.Name,
			&m.Parent,
			&m.Version,
			&m.Description,
			&m.UploadIds,
		); err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}

	return ms, nil

}
