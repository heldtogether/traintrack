package uploads

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type mockRepo struct {
	createFunc func(upload *Upload) (*Upload, error)
}

func (m *mockRepo) Create(upload *Upload) (*Upload, error) {
	if m.createFunc != nil {
		return m.createFunc(upload)
	}
	return upload, nil
}

type mockStorage struct {
	saveFileFn func(dst string, file multipart.File) error
}

func (m *mockStorage) SaveFile(dst string, file multipart.File) error {
	return m.saveFileFn(dst, file)
}

func newMultipartForm(t *testing.T, field, filename, content string) (*http.Request, string) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, err = io.Copy(fw, strings.NewReader(content))
	if err != nil {
		t.Fatalf("failed to write form content: %v", err)
	}
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/uploads", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req, content
}

func TestUploadsHandler(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		requestSetup     func(t *testing.T) *http.Request
		createUploadFn   func(upload *Upload) (*Upload, error)
		saveFileFn       func(dst string, file multipart.File) error
		expectedStatus   int
		expectedContains string
	}{
		{
			name:   "POST success",
			method: http.MethodPost,
			requestSetup: func(t *testing.T) *http.Request {
				req, _ := newMultipartForm(t, "files", "test.txt", "hello")
				return req
			},
			createUploadFn: func(upload *Upload) (*Upload, error) {
				upload.ID = "1"
				return upload, nil
			},
			saveFileFn: func(dst string, file multipart.File) error {
				return nil
			},
			expectedStatus:   http.StatusCreated,
			expectedContains: `{"id": "1", "files": [{"provider": "filesystem", "filename": "test.txt", "path": "tmp/uploads/mock-id/"}]}`,
		},
		{
			name:   "POST success",
			method: http.MethodPost,
			requestSetup: func(t *testing.T) *http.Request {
				req, _ := newMultipartForm(t, "files", "test.txt", "hello")
				return req
			},
			createUploadFn: func(upload *Upload) (*Upload, error) {
				return nil, errors.New("boom")
			},
			saveFileFn: func(dst string, file multipart.File) error {
				return nil
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedContains: `{"code": 500, "error": "Failed to create upload", "reason": "boom"}`,
		},
		{
			name:   "POST failure - parse error",
			method: http.MethodPost,
			requestSetup: func(t *testing.T) *http.Request {
				body := strings.NewReader("not a real multipart body")
				req := httptest.NewRequest(http.MethodPost, "/uploads", body)
				req.Header.Set("Content-Type", "multipart/form-data; boundary=badboundary")
				return req
			},
			expectedStatus:   http.StatusBadRequest,
			expectedContains: `{"code": 400, "error": "Failed to create upload", "reason": "multipart: NextPart: EOF"}`,
		},
		{
			name:   "POST failure - no files",
			method: http.MethodPost,
			requestSetup: func(t *testing.T) *http.Request {
				var b bytes.Buffer
				w := multipart.NewWriter(&b)
				w.Close()
				req := httptest.NewRequest(http.MethodPost, "/uploads", &b)
				req.Header.Set("Content-Type", w.FormDataContentType())
				return req
			},
			expectedStatus:   http.StatusBadRequest,
			expectedContains: `{"code": 400, "error": "Failed to create upload", "reason": "no files uploaded"}`,
		},
		{
			name:   "POST failure - save file failed",
			method: http.MethodPost,
			requestSetup: func(t *testing.T) *http.Request {
				req, _ := newMultipartForm(t, "files", "bad.txt", "fail")
				return req
			},
			saveFileFn: func(dst string, file multipart.File) error {
				return errors.New("upload failed")
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedContains: `{"code": 500, "error": "Failed to create upload", "reason": "upload failed"}`,
		},
		{
			name:   "METHOD failure",
			method: http.MethodDelete,
			requestSetup: func(t *testing.T) *http.Request {
				return httptest.NewRequest(http.MethodDelete, "/uploads", nil)
			},
			expectedStatus:   http.StatusMethodNotAllowed,
			expectedContains: `{"code": 405, "error": "Method not allowed", "reason": ""}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var storage Storage
			if tc.saveFileFn != nil {
				storage = &mockStorage{saveFileFn: tc.saveFileFn}
			} else {
				storage = &mockStorage{saveFileFn: func(dst string, file multipart.File) error {
					return nil
				}}
			}

			handler := NewHandler(
				&mockRepo{
					createFunc: tc.createUploadFn,
				},
				storage,
				func() string {
					return "mock-id"
				},
			)
			req := tc.requestSetup(t)
			rr := httptest.NewRecorder()
			handler.Uploads(rr, req)

			checkResponse(t, rr.Result(), tc.expectedStatus, tc.expectedContains)
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
		t.Errorf("status mismatch - wanted %d, got %d", expectedStatus, got.StatusCode)
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
