package models

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Model struct {
	ID          string  `json:"id"`
	Name        string  `json:"name" validate:"required"`
	Parent      *string `json:"parent"`
	Version     string  `json:"version" validate:"required"`
	Description string  `json:"description" validate:"required"`

	UploadIds map[string]string `json:"artefacts"`

	DatasetId string `json:"dataset"`

	Config      json.RawMessage `json:"config"`
	Metadata    json.RawMessage `json:"metadata"`
	Environment json.RawMessage `json:"environment"`
	Evaluation  json.RawMessage `json:"evaluation"`
}

func (m *Model) GetID() string          { return m.ID }
func (m *Model) GetName() string        { return m.Name }
func (m *Model) GetDescription() string { return m.Description }
func (m *Model) GetVersion() string     { return m.Version }
func (m *Model) GetParent() *string     { return m.Parent }

const (
	createQuery = `INSERT INTO 
models (name, parent, version, description, dataset, config, metadata, environment, evaluation) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	listQuery = `SELECT 
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

type Store struct {
	q Querier
}

func NewStore(q Querier) *Store {
	return &Store{
		q: q,
	}
}

// Don't export, we only want people using the designated
// creator struct to ensure that the business logic is followed.
func (s *Store) create(d *Model) (*Model, error) {
	return s.createWithQuerier(s.q, d)
}

// Don't export, we only want people using the designated
// creator struct to ensure that the business logic is followed.
func (s *Store) createWithQuerier(conn Querier, m *Model) (*Model, error) {
	query := createQuery
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

/*
List returns a list of all known Models.
*/
func (s *Store) List() ([]*Model, error) {
	rows, err := s.q.Query(
		context.TODO(),
		listQuery,
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
