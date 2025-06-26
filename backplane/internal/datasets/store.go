package datasets

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Dataset struct {
	ID          string  `json:"id"`
	Name        string  `json:"name" validate:"required"`
	Parent      *string `json:"parent"`
	Version     string  `json:"version" validate:"required"`
	Description string  `json:"description" validate:"required"`

	UploadIds map[string]string `json:"artefacts"`
}

func (m *Dataset) GetID() string          { return m.ID }
func (m *Dataset) GetName() string        { return m.Name }
func (m *Dataset) GetDescription() string { return m.Description }
func (m *Dataset) GetVersion() string     { return m.Version }
func (m *Dataset) GetParent() *string     { return m.Parent }

const (
	createQuery = `INSERT INTO datasets 
(name, parent, version, description) 
VALUES ($1, $2, $3, $4) 
RETURNING id`
	listQuery = `SELECT 
  d.id,
  d.name,
  d.parent,
  d.version,
  d.description,
  COALESCE(
    jsonb_object_agg(file_key, u.id) FILTER (WHERE file_key IS NOT NULL),
    '{}'::jsonb
  ) AS artefacts
FROM datasets d
LEFT JOIN uploads u ON u.dataset_id = d.id
LEFT JOIN LATERAL jsonb_object_keys(u.files) AS file_key ON true
GROUP BY d.id, d.name, d.parent, d.version, d.description;`
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
func (s *Store) create(d *Dataset) (*Dataset, error) {
	return s.createWithQuerier(s.q, d)
}

// Don't export, we only want people using the designated
// creator struct to ensure that the business logic is followed.
func (s *Store) createWithQuerier(q Querier, d *Dataset) (*Dataset, error) {
	query := createQuery
	row := q.QueryRow(
		context.Background(),
		query,
		d.Name,
		d.Parent,
		d.Version,
		d.Description,
	)

	var id string
	if err := row.Scan(&id); err != nil {
		return nil, err
	}

	return &Dataset{
		ID:          id,
		Name:        d.Name,
		Parent:      d.Parent,
		Version:     d.Version,
		Description: d.Description,
	}, nil
}

/*
List returns a list of all known Datasets.
*/
func (s *Store) List() ([]*Dataset, error) {
	rows, err := s.q.Query(
		context.TODO(),
		listQuery,
	)
	if err != nil {
		return nil, fmt.Errorf("could not query datasets: %s", err)
	}

	defer rows.Close()

	ds := []*Dataset{}
	for rows.Next() {
		d := &Dataset{}
		if err := rows.Scan(
			&d.ID,
			&d.Name,
			&d.Parent,
			&d.Version,
			&d.Description,
			&d.UploadIds,
		); err != nil {
			return nil, err
		}
		ds = append(ds, d)
	}

	return ds, nil

}
