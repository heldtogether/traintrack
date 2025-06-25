package models

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/heldtogether/traintrack/internal/uploads"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

type MockUploadsRepo struct {
	GetByIDFunc func(ctx context.Context, id string) (*uploads.Upload, error)
	MoveFunc    func(ctx context.Context, u *uploads.Upload) error
}

func (m *MockUploadsRepo) GetByIDWithQuerier(_ uploads.Querier, id string) (*uploads.Upload, error) {
	return m.GetByIDFunc(context.Background(), id)
}
func (m *MockUploadsRepo) MoveWithQuerier(_ uploads.Querier, u *uploads.Upload) error {
	return m.MoveFunc(context.Background(), u)
}

type MockModelsRepo struct {
	CreateFunc func(ctx context.Context, d *Model) (*Model, error)
	ListFunc   func() ([]*Model, error)
}

func (m *MockModelsRepo) CreateWithQuerier(_ Querier, d *Model) (*Model, error) {
	return m.CreateFunc(context.Background(), d)
}

func (m *MockModelsRepo) List() ([]*Model, error) {
	return m.ListFunc()
}

type mockDB struct {
	tx pgx.Tx
}

func (m *mockDB) Begin(ctx context.Context) (pgx.Tx, error) {
	return m.tx, nil
}

type loggingTx struct {
	pgx.Tx
	log            *[]string
	commitOverride func(ctx context.Context) error // optional
}

func (l *loggingTx) Commit(ctx context.Context) error {
	if l.commitOverride != nil {
		return l.commitOverride(ctx)
	}
	*l.log = append(*l.log, "commit")
	return nil
}

func (l *loggingTx) Rollback(ctx context.Context) error {
	*l.log = append(*l.log, "rollback")
	return nil
}

type MockStorage struct {
	MoveFunc func(src, dst string) error
}

func (m *MockStorage) MoveFile(src, dst string) error {
	return m.MoveFunc(src, dst)
}

type MockTx struct {
	CommitFunc   func(ctx context.Context) error
	RollbackFunc func(ctx context.Context) error
}

func (m *MockTx) Commit(ctx context.Context) error   { return m.CommitFunc(ctx) }
func (m *MockTx) Rollback(ctx context.Context) error { return m.RollbackFunc(ctx) }
func (m *MockTx) Query(ctx context.Context, q string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (m *MockTx) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	return nil
}
func (m *MockTx) Exec(ctx context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func TestService_Create(t *testing.T) {
	modelID := "ds456"
	uploadID := "upload123"
	fileName := "artifact.txt"

	tests := []struct {
		name              string
		failCreate        bool
		failGetUpload     bool
		failMoveFile      bool
		failMoveUpload    bool
		failCommit        bool
		wantCalled        []string
		expectCreateError bool
	}{
		{
			name: "success",
			wantCalled: []string{
				"create-model",
				"get-upload",
				"move-file temp/path/artifact.txt -> models/ds456/artifact.txt",
				"move-upload",
				"commit",
			},
		},
		{
			name:              "create fails",
			failCreate:        true,
			wantCalled:        []string{"create-model", "rollback"},
			expectCreateError: true,
		},
		{
			name:              "get upload fails",
			failGetUpload:     true,
			wantCalled:        []string{"create-model", "get-upload", "rollback"},
			expectCreateError: true,
		},
		{
			name:              "move file fails",
			failMoveFile:      true,
			wantCalled:        []string{"create-model", "get-upload", "move-file temp/path/artifact.txt -> models/ds456/artifact.txt", "rollback"},
			expectCreateError: true,
		},
		{
			name:              "move upload fails",
			failMoveUpload:    true,
			wantCalled:        []string{"create-model", "get-upload", "move-file temp/path/artifact.txt -> models/ds456/artifact.txt", "move-upload", "rollback"},
			expectCreateError: true,
		},
		{
			name:              "commit fails",
			failCommit:        true,
			wantCalled:        []string{"create-model", "get-upload", "move-file temp/path/artifact.txt -> models/ds456/artifact.txt", "move-upload", "commit", "rollback"},
			expectCreateError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var called []string

			mockPgx, _ := pgxmock.NewConn()
			baseTx, _ := mockPgx.Begin(context.Background())

			// wrap the tx so we can simulate commit failure and log the call
			tx := &loggingTx{
				Tx:  baseTx,
				log: &called,
				commitOverride: func(ctx context.Context) error {
					called = append(called, "commit")
					if tc.failCommit {
						return errors.New("commit boom")
					}
					return nil
				},
			}

			mockDB := &mockDB{tx: tx}

			mockModelRepo := &MockModelsRepo{
				CreateFunc: func(ctx context.Context, d *Model) (*Model, error) {
					called = append(called, "create-model")
					if tc.failCreate {
						return nil, errors.New("boom")
					}
					return &Model{
						ID:        modelID,
						UploadIds: d.UploadIds,
					}, nil
				},
			}

			mockUploadRepo := &MockUploadsRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*uploads.Upload, error) {
					called = append(called, "get-upload")
					if tc.failGetUpload {
						return nil, errors.New("boom")
					}
					return &uploads.Upload{
						ID: uploadID,
						Files: map[string]uploads.FileRef{
							"artefact": {
								Provider: uploads.ProviderFileSystem,
								FileName: fileName,
								Path:     "temp/path/",
							}},
					}, nil
				},
				MoveFunc: func(ctx context.Context, u *uploads.Upload) error {
					called = append(called, "move-upload")
					if tc.failMoveUpload {
						return errors.New("boom")
					}
					return nil
				},
			}

			mockStorage := &MockStorage{
				MoveFunc: func(src, dst string) error {
					called = append(called, fmt.Sprintf("move-file %s -> %s", src, dst))
					if tc.failMoveFile {
						return errors.New("boom")
					}
					return nil
				},
			}

			service := &Service{
				ModelsRepo:  mockModelRepo,
				UploadsRepo: mockUploadRepo,
				Storage:     mockStorage,
				DB:          mockDB,
			}

			ctx := context.Background()
			_, err := service.Create(ctx, &Model{UploadIds: map[string]string{"file1": uploadID}})

			if tc.expectCreateError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectCreateError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for i, step := range tc.wantCalled {
				if i >= len(called) || called[i] != step {
					t.Errorf("step %d: got %q, want %q", i, called[i], step)
				}
			}

			if len(called) != len(tc.wantCalled) {
				t.Errorf("called steps = %v, want = %v", called, tc.wantCalled)
			}
		})
	}
}

func TestService_List(t *testing.T) {
	want := []*Model{
		{ID: "1", Name: "Training Set"},
		{ID: "2", Name: "Validation Set"},
	}

	mockRepo := &MockModelsRepo{
		ListFunc: func() ([]*Model, error) {
			return want, nil
		},
	}
	svc := &Service{ModelsRepo: mockRepo}

	got, err := svc.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
