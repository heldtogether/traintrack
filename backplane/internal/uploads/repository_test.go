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

func TestMove(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	upload := &Upload{
		ID: "1",
		Files: []FileRef{
			{Provider: ProviderFileSystem, FileName: "file", Path: "/"},
		},
	}
	filesJSON, err := json.Marshal(upload.Files)
	if err != nil {
		t.Fatal(err)
	}

	db.ExpectExec(regexp.QuoteMeta(UpdateQuery)).
		WithArgs(filesJSON, upload.ID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(db)

	err = repo.Move(upload)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestGetByIDWithQuerier(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	want := &Upload{
		ID: "123",
		Files: []FileRef{
			{Provider: ProviderFileSystem, FileName: "report.pdf", Path: "/docs"},
		},
	}
	filesJSON, err := json.Marshal(want.Files)
	if err != nil {
		t.Fatal(err)
	}

	db.ExpectQuery(regexp.QuoteMeta(GetQuery)).
		WithArgs("123").
		WillReturnRows(pgxmock.NewRows([]string{"id", "files"}).
			AddRow(want.ID, filesJSON),
		)

	repo := NewRepository(nil)
	got, err := repo.GetByIDWithQuerier(db, "123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestGetByIDWithQuerier_ScanError(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.ExpectQuery(regexp.QuoteMeta(GetQuery)).
		WithArgs("999").
		WillReturnError(errors.New("scan fail"))

	repo := NewRepository(nil)
	_, err = repo.GetByIDWithQuerier(db, "999")
	if err == nil || !strings.Contains(err.Error(), "scan") {
		t.Fatalf("expected scan error, got %v", err)
	}
}

func TestGetByIDWithQuerier_UnmarshalError(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	invalidJSON := []byte(`{"bad":`) // Invalid JSON

	db.ExpectQuery(regexp.QuoteMeta(GetQuery)).
		WithArgs("123").
		WillReturnRows(pgxmock.NewRows([]string{"id", "files"}).
			AddRow("123", invalidJSON),
		)

	repo := NewRepository(nil)
	_, err = repo.GetByIDWithQuerier(db, "123")
	if err == nil || !strings.Contains(err.Error(), "unmarshal") {
		t.Fatalf("expected unmarshal error, got %v", err)
	}
}
