package datasets

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

type mockCreatorAndLister struct {
	CreateFn func(ctx context.Context, d *Dataset) (*Dataset, error)
	ListFn   func() ([]*Dataset, error)
}

func (m *mockCreatorAndLister) Create(ctx context.Context, d *Dataset) (*Dataset, error) {
	return m.CreateFn(ctx, d)
}

func (m *mockCreatorAndLister) List() ([]*Dataset, error) {
	return m.ListFn()
}

func TestRouter(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		body             string
		listDatasetsFn   func() ([]*Dataset, error)
		createDatasetFn  func(ctx context.Context, d *Dataset) (*Dataset, error)
		expectedStatus   int
		expectedContains string
	}{
		{
			name:   "GET success",
			method: http.MethodGet,
			listDatasetsFn: func() ([]*Dataset, error) {
				return []*Dataset{{ID: "1", UploadIds: map[string]string{"file1": "abc"}}}, nil
			},
			expectedStatus:   http.StatusOK,
			expectedContains: `[{"id": "1", "name": "", "parent": null, "version":"", "description":"", "artefacts": {"file1": "abc"}}]`,
		},
		{
			name:   "GET failure",
			method: http.MethodGet,
			listDatasetsFn: func() ([]*Dataset, error) {
				return nil, errors.New("boom")
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedContains: `{"code": 500, "error": "Failed to list datasets", "reason": "boom"}`,
		},
		{
			name:           "POST success",
			method:         http.MethodPost,
			body:           `{"id": "", "name": "name", "parent": null, "version": "version", "description": "description"}`,
			listDatasetsFn: nil,
			createDatasetFn: func(_ context.Context, r *Dataset) (*Dataset, error) {
				return &Dataset{ID: "123", Name: "name", Parent: nil, Version: "version", Description: "description", UploadIds: map[string]string{}}, nil
			},
			expectedStatus:   http.StatusCreated,
			expectedContains: `{"id": "123", "name": "name", "parent": null, "version": "version", "description": "description", "artefacts": {}}`,
		},
		{
			name:           "POST failure - unparseable request",
			method:         http.MethodPost,
			body:           ``,
			listDatasetsFn: nil,
			createDatasetFn: func(_ context.Context, r *Dataset) (*Dataset, error) {
				return nil, errors.New("bad request")
			},
			expectedStatus:   http.StatusBadRequest,
			expectedContains: `{"code": 400, "error": "Failed to create dataset", "reason": "could not parse body: EOF"}`,
		},
		{
			name:           "POST failure - failed validation",
			method:         http.MethodPost,
			body:           `{"id": ""}`,
			listDatasetsFn: nil,
			createDatasetFn: func(_ context.Context, r *Dataset) (*Dataset, error) {
				return nil, errors.New("bad request")
			},
			expectedStatus: http.StatusBadRequest,
			expectedContains: `{"code": 400, "error": "Failed to create dataset", "reason": "bad input", "details": {
				"name": "name is a required field",
				"description": "description is a required field",
				"version": "version is a required field"
			}}`,
		},
		{
			name:           "POST failure - service failed",
			method:         http.MethodPost,
			body:           `{"id": "", "name": "name", "parent": null, "version":"version", "description":"description"}`,
			listDatasetsFn: nil,
			createDatasetFn: func(_ context.Context, r *Dataset) (*Dataset, error) {
				return nil, errors.New("boom")
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedContains: `{"code": 500, "error": "Failed to create dataset", "reason": "boom"}`,
		},
		{
			name:             "METHOD failure",
			method:           http.MethodTrace,
			body:             ``,
			listDatasetsFn:   nil,
			createDatasetFn:  nil,
			expectedStatus:   http.StatusMethodNotAllowed,
			expectedContains: `{"code": 405, "error": "Method not allowed", "reason": ""}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := &mockCreatorAndLister{
				CreateFn: tc.createDatasetFn,
				ListFn:   tc.listDatasetsFn,
			}
			handler := NewHandler(mockService, mockService)

			var bodyReader io.Reader
			if tc.body != "" {
				bodyReader = strings.NewReader(tc.body)
			}

			req := httptest.NewRequest(tc.method, "/datasets", bodyReader)
			rr := httptest.NewRecorder()

			handler.Datasets(rr, req)

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
