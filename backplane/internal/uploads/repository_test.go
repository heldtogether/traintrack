package uploads

import (
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
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
		Files: map[string]FileRef{
			"artefact": {Provider: ProviderFileSystem, FileName: "test", Path: "/"},
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
			Files: map[string]FileRef{
				"artefact": {Provider: ProviderFileSystem, FileName: "test", Path: "/"},
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

	files := map[string]FileRef{
		"artefact": {Provider: ProviderFileSystem, FileName: "test", Path: "/"},
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
		Files: map[string]FileRef{
			"artefact": {Provider: ProviderFileSystem, FileName: "file", Path: "/"},
		},
		DatasetID: pointerTo("d123"),
	}
	filesJSON, err := json.Marshal(upload.Files)
	if err != nil {
		t.Fatal(err)
	}

	db.ExpectExec(regexp.QuoteMeta(UpdateQuery)).
		WithArgs(filesJSON, upload.DatasetID, upload.ID).
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
		Files: map[string]FileRef{
			"artefact": {Provider: ProviderFileSystem, FileName: "report.pdf", Path: "/docs"},
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

func TestRepository_Get_Success(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	expectedID := "abc-123"
	expectedFiles := map[string]FileRef{
		"artefact": {Provider: "filesystem", FileName: "file1.txt", Path: "uploads/abc-123"},
	}

	rows := pgxmock.NewRows([]string{"id", "files"}).
		AddRow(expectedID, expectedFiles)

	db.ExpectQuery(regexp.QuoteMeta(GetQuery)).
		WithArgs(expectedID).
		WillReturnRows(rows)

	repo := &Repository{conn: db}
	upload, err := repo.Get(expectedID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if upload.ID != expectedID {
		t.Errorf("expected ID %s, got %s", expectedID, upload.ID)
	}

	if !reflect.DeepEqual(upload.Files, expectedFiles) {
		t.Errorf("expected Files %+v, got %+v", expectedFiles, upload.Files)
	}
}

func TestRepository_Get_ScanError(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.ExpectQuery(regexp.QuoteMeta(GetQuery)).
		WithArgs("999").
		WillReturnError(errors.New("scan fail"))

	repo := &Repository{conn: db}
	_, err = repo.Get("999")
	if err == nil || !strings.Contains(err.Error(), "scan fail") {
		t.Fatalf("expected scan fail error, got %v", err)
	}
}

func TestRepository_Get_NotFound(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.ExpectQuery(regexp.QuoteMeta(GetQuery)).
		WithArgs("123").
		WillReturnError(pgx.ErrNoRows)

	repo := &Repository{conn: db}
	_, err = repo.Get("123")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected pgx.ErrNoRows, got %v", err)
	}
}
