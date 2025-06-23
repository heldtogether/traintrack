package datasets

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const (
	CreateQuery = `INSERT INTO datasets (name, parent, version, description) VALUES ($1, $2, $3, $4) RETURNING id`
	ListQuery   = `SELECT 
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

type Repository struct {
	conn Querier
}

func NewRepository(conn Querier) *Repository {
	return &Repository{
		conn: conn,
	}
}

func (r *Repository) Create(d *Dataset) (*Dataset, error) {
	return r.CreateWithQuerier(r.conn, d)
}

func (r *Repository) CreateWithQuerier(conn Querier, d *Dataset) (*Dataset, error) {
	query := CreateQuery
	row := conn.QueryRow(
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

func (r *Repository) List() ([]*Dataset, error) {
	rows, err := r.conn.Query(
		context.TODO(),
		ListQuery,
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
