package datasets

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

var nilStr *string

func TestList(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rows := db.NewRows([]string{"id", "name", "parent", "version", "description", "artefacts"}).
		AddRow("1", "", nil, "", "", make(map[string]string))

	db.ExpectQuery(
		regexp.QuoteMeta(ListQuery),
	).
		WillReturnRows(rows)

	service := NewRepository(db)

	ds, err := service.List()
	if err != nil {
		t.Errorf("could not list: %s", err)
	}
	if ds == nil {
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
		regexp.QuoteMeta(ListQuery),
	).
		WillReturnError(fmt.Errorf("expected error"))

	service := NewRepository(db)

	ds, err := service.List()
	if err == nil {
		t.Errorf("could not list: %s", err)
	}
	if ds != nil {
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
		regexp.QuoteMeta(ListQuery),
	).
		WillReturnRows(rows)

	service := NewRepository(db)

	ds, err := service.List()
	if err == nil {
		t.Errorf("expected error from Scan, got nil")
	}
	if ds != nil {
		t.Errorf("expected scan error, got: %+v", ds)
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
		regexp.QuoteMeta(CreateQuery),
	).
		WithArgs("name", nilStr, "1.0.0", "description").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("1"))

	service := NewRepository(db)

	want := &Dataset{
		ID:          "1",
		Name:        "name",
		Parent:      nil,
		Version:     "1.0.0",
		Description: "description",
	}
	got, err := service.Create(
		&Dataset{
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
		regexp.QuoteMeta(CreateQuery),
	).
		WithArgs("name", nilStr, "1.0.0", "description").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(unscannable{}))

	service := NewRepository(db)

	got, err := service.Create(
		&Dataset{
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
