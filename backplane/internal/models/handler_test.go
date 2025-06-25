package models

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type mockService struct {
	CreateFn func(ctx context.Context, m *Model) (*Model, error)
	ListFn   func() ([]*Model, error)
}

func (m *mockService) Create(ctx context.Context, d *Model) (*Model, error) {
	return m.CreateFn(ctx, d)
}

func (m *mockService) List() ([]*Model, error) {
	return m.ListFn()
}

func TestRouter(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		body             string
		listModelsFn     func() ([]*Model, error)
		createModelFn    func(ctx context.Context, m *Model) (*Model, error)
		expectedStatus   int
		expectedContains string
	}{
		{
			name:   "GET success",
			method: http.MethodGet,
			listModelsFn: func() ([]*Model, error) {
				return []*Model{{ID: "1", UploadIds: map[string]string{"file1": "abc"}}}, nil
			},
			expectedStatus:   http.StatusOK,
			expectedContains: `[{"id": "1", "name": "", "parent": null, "version":"", "description":"", "artefacts": {"file1": "abc"}, "config":null, "environment":null, "evaluation": null, "metadata": null, "dataset": ""}]`,
		},
		{
			name:   "GET failure",
			method: http.MethodGet,
			listModelsFn: func() ([]*Model, error) {
				return nil, errors.New("boom")
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedContains: `{"code": 500, "error": "Failed to list models", "reason": "boom"}`,
		},
		{
			name:         "POST success",
			method:       http.MethodPost,
			body:         `{"id": "", "name": "name", "parent": null, "version": "version", "description": "description"}`,
			listModelsFn: nil,
			createModelFn: func(_ context.Context, r *Model) (*Model, error) {
				return &Model{ID: "123", Name: "name", Parent: nil, Version: "version", Description: "description", UploadIds: map[string]string{}}, nil
			},
			expectedStatus:   http.StatusCreated,
			expectedContains: `{"id": "123", "name": "name", "parent": null, "version": "version", "description": "description", "artefacts": {}, "config":null, "environment":null, "evaluation": null, "metadata": null, "dataset": ""}`,
		},
		{
			name:         "POST failure - unparseable request",
			method:       http.MethodPost,
			body:         ``,
			listModelsFn: nil,
			createModelFn: func(_ context.Context, r *Model) (*Model, error) {
				return nil, errors.New("bad request")
			},
			expectedStatus:   http.StatusBadRequest,
			expectedContains: `{"code": 400, "error": "Failed to create model", "reason": "could not parse body: EOF"}`,
		},
		{
			name:         "POST failure - failed validation",
			method:       http.MethodPost,
			body:         `{"id": ""}`,
			listModelsFn: nil,
			createModelFn: func(_ context.Context, r *Model) (*Model, error) {
				return nil, errors.New("bad request")
			},
			expectedStatus: http.StatusBadRequest,
			expectedContains: `{"code": 400, "error": "Failed to create model", "reason": "bad input", "details": {
				"name": "name is a required field",
				"description": "description is a required field",
				"version": "version is a required field"
			}}`,
		},
		{
			name:         "POST failure - service failed",
			method:       http.MethodPost,
			body:         `{"id": "", "name": "name", "parent": null, "version":"version", "description":"description"}`,
			listModelsFn: nil,
			createModelFn: func(_ context.Context, r *Model) (*Model, error) {
				return nil, errors.New("boom")
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedContains: `{"code": 500, "error": "Failed to create model", "reason": "boom"}`,
		},
		{
			name:             "METHOD failure",
			method:           http.MethodTrace,
			body:             ``,
			listModelsFn:     nil,
			createModelFn:    nil,
			expectedStatus:   http.StatusMethodNotAllowed,
			expectedContains: `{"code": 405, "error": "Method not allowed", "reason": ""}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := &mockService{
				CreateFn: tc.createModelFn,
				ListFn:   tc.listModelsFn,
			}
			handler := NewHandler(mockService)

			var bodyReader io.Reader
			if tc.body != "" {
				bodyReader = strings.NewReader(tc.body)
			}

			req := httptest.NewRequest(tc.method, "/models", bodyReader)
			rr := httptest.NewRecorder()

			handler.Models(rr, req)

			res := rr.Result()
			checkResponse(t, res, tc.expectedStatus, tc.expectedContains)
		})
	}
}

func checkResponse(t *testing.T, got *http.Response, expectedStatus int, expected string) {

	defer got.Body.Close()

	body, err := io.ReadAll(got.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %s", err)
	}

	if expectedStatus != got.StatusCode {
		t.Errorf("status mismatch - wanted %d, got %d",
			expectedStatus,
			got.StatusCode,
		)
	}

	var gotData any
	if err := json.Unmarshal(body, &gotData); err != nil {
		t.Fatalf("failed to unmarshal response body: %v\nbody: %s", err, string(body))
	}

	var expectedData any
	if err := json.Unmarshal([]byte(expected), &expectedData); err != nil {
		t.Fatalf("failed to unmarshal expected value: %v\njson: %s", err, string(expected))
	}

	if !reflect.DeepEqual(expectedData, gotData) {
		t.Errorf("JSON mismatch:\nexpected: %+v\ngot: %+v", expectedData, gotData)
	}
}
