package models

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

var nilStr *string
var nilJSONBlob json.RawMessage

func TestList(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rows := db.NewRows([]string{"id", "name", "parent", "version", "description", "artefacts"}).
		AddRow("1", "", nil, "", "", map[string]string{})

	db.ExpectQuery(
		regexp.QuoteMeta(listQuery),
	).
		WillReturnRows(rows)

	service := NewStore(db)

	ms, err := service.List()
	if err != nil {
		t.Errorf("could not list: %s", err)
	}
	if ms == nil {
		t.Errorf("could not list, nil returned")
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestListFailOnQuery(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.ExpectQuery(
		regexp.QuoteMeta(listQuery),
	).
		WillReturnError(fmt.Errorf("expected error"))

	service := NewStore(db)

	ms, err := service.List()
	if err == nil {
		t.Errorf("could not list: %s", err)
	}
	if ms != nil {
		t.Errorf("could not list, nil returned")
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestListFailOnScan(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rows := db.NewRows([]string{"name"}).
		AddRow(nil)

	db.ExpectQuery(
		regexp.QuoteMeta(listQuery),
	).
		WillReturnRows(rows)

	service := NewStore(db)

	ms, err := service.List()
	if err == nil {
		t.Errorf("expected error from Scan, got nil")
	}
	if ms != nil {
		t.Errorf("expected scan error, got: %+v", ms)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCreate(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.ExpectQuery(
		regexp.QuoteMeta(createQuery),
	).
		WithArgs(
			"name",
			nilStr,
			"1.0.0",
			"description",
			"",
			nilJSONBlob,
			nilJSONBlob,
			nilJSONBlob,
			nilJSONBlob,
		).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("1"))

	service := NewStore(db)

	want := &Model{
		ID:          "1",
		Name:        "name",
		Parent:      nil,
		Version:     "1.0.0",
		Description: "description",
	}
	got, err := service.create(
		&Model{
			Name:        "name",
			Parent:      nil,
			Version:     "1.0.0",
			Description: "description",
		})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, wanted %+v", got, want)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCreateFailOnScan(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	type unscannable struct{}

	db.ExpectQuery(
		regexp.QuoteMeta(createQuery),
	).
		WithArgs(
			"name",
			nilStr,
			"1.0.0",
			"description",
			"",
			nilJSONBlob,
			nilJSONBlob,
			nilJSONBlob,
			nilJSONBlob,
		).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(unscannable{}))

	service := NewStore(db)

	got, err := service.create(
		&Model{
			Name:        "name",
			Parent:      nil,
			Version:     "1.0.0",
			Description: "description",
		})
	if err == nil {
		t.Errorf("expected error from Scan, got nil")
	}
	if got != nil {
		t.Errorf("expected nil on scan error, got: %+v", got)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
