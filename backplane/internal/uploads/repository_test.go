package uploads

import (
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestCreate(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	want := &Upload{
		ID: "1",
		Files: []FileRef{
			{Provider: ProviderFileSystem, FileName: "test", Path: "/"},
		},
	}

	filesJSON, err := json.Marshal(want.Files)
	if err != nil {
		t.Fatal(err)
	}

	db.ExpectQuery(
		regexp.QuoteMeta(CreateQuery),
	).
		WithArgs(filesJSON).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("1"))

	service := NewRepository(db)

	got, err := service.Create(
		&Upload{
			Files: []FileRef{
				{Provider: ProviderFileSystem, FileName: "test", Path: "/"},
			},
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

func TestCreate_ScanFails(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	files := []FileRef{
		{Provider: ProviderFileSystem, FileName: "test", Path: "/"},
	}
	filesJSON, err := json.Marshal(files)
	if err != nil {
		t.Fatal(err)
	}

	db.ExpectQuery(regexp.QuoteMeta(CreateQuery)).
		WithArgs(filesJSON).
		WillReturnError(errors.New("scan failed"))

	repo := NewRepository(db)
	_, err = repo.Create(&Upload{Files: files})

	if err == nil || !strings.Contains(err.Error(), "scan failed") {
		t.Fatalf("expected scan error, got %v", err)
	}
}

