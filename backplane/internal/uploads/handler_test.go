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

	"github.com/gorilla/mux"
)

type mockRepo struct {
	createFunc func(upload *Upload) (*Upload, error)
	getFunc    func(id string) (*Upload, error)
}

func (m *mockRepo) Create(upload *Upload) (*Upload, error) {
	if m.createFunc != nil {
		return m.createFunc(upload)
	}
	return upload, nil
}

func (m *mockRepo) Get(id string) (*Upload, error) {
	if m.getFunc != nil {
		return m.getFunc(id)
	}
	return nil, errors.New("mock: no file available")
}

type mockStorage struct {
	saveFileFn func(dst string, file multipart.File) error
	readFileFn func(path string) ([]byte, error)
}

func (m *mockStorage) SaveFile(dst string, file multipart.File) error {
	return m.saveFileFn(dst, file)
}

func (m *mockStorage) ReadFile(path string) ([]byte, error) {
	return m.readFileFn(path)
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
		getUploadFn      func(id string) (*Upload, error)
		saveFileFn       func(dst string, file multipart.File) error
		readFileFn       func(id string) ([]byte, error)
		expectedStatus   int
		expectedContains string
		expectRaw        *bool
	}{
		{
			name:   "POST success",
			method: http.MethodPost,
			requestSetup: func(t *testing.T) *http.Request {
				req, _ := newMultipartForm(t, "artefact", "test.txt", "hello")
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
			expectedContains: `{"id": "1", "files": {"artefact": {"provider": "filesystem", "filename": "test.txt", "path": "tmp/uploads/mock-id/"}}}`,
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
			name:   "GET returns raw file bytes",
			method: http.MethodGet,
			requestSetup: func(t *testing.T) *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/uploads/mock-id/artefact", nil)
				return req
			},
			createUploadFn: nil,
			getUploadFn: func(id string) (*Upload, error) {
				return &Upload{
					ID: id,
					Files: map[string]FileRef{
						"artefact": {
							FileName: "test.txt",
							Path:     "mock-id",
						},
					},
				}, nil
			},
			saveFileFn: nil,
			readFileFn: func(path string) ([]byte, error) {
				return []byte("hello world"), nil
			},
			expectedStatus:   http.StatusOK,
			expectedContains: "hello world",
			expectRaw:        pointerTo(true),
		},
		{
			name:   "GET to unknown upload returns error",
			method: http.MethodGet,
			requestSetup: func(t *testing.T) *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/uploads/unknown-id/artefact", nil)
				return req
			},
			createUploadFn: nil,
			getUploadFn: func(id string) (*Upload, error) {
				return nil, errors.New("not found")
			},
			saveFileFn: nil,
			readFileFn: func(path string) ([]byte, error) {
				return nil, errors.New("unexpected id")
			},
			expectedStatus:   http.StatusNotFound,
			expectedContains: `{"code": 404, "error": "Upload not found", "reason": "not found"}`,
		},
		{
			name:   "GET to unknown file returns error",
			method: http.MethodGet,
			requestSetup: func(t *testing.T) *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/uploads/mock-id/unknown", nil)
				return req
			},
			createUploadFn: nil,
			getUploadFn: func(id string) (*Upload, error) {
				return &Upload{
					ID: id,
					Files: map[string]FileRef{
						"artefact": {
							FileName: "test.txt",
							Path:     "mock-id",
						},
					},
				}, nil
			},
			saveFileFn: nil,
			readFileFn: func(path string) ([]byte, error) {
				return nil, errors.New("unexpected id")
			},
			expectedStatus:   http.StatusNotFound,
			expectedContains: `{"code": 404, "error": "File not found", "reason": "unknown file"}`,
		},
		{
			name:   "GET to problematic file returns error",
			method: http.MethodGet,
			requestSetup: func(t *testing.T) *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/uploads/mock-id/artefact", nil)
				return req
			},
			createUploadFn: nil,
			getUploadFn: func(id string) (*Upload, error) {
				return &Upload{
					ID: id,
					Files: map[string]FileRef{
						"artefact": {
							FileName: "test.txt",
							Path:     "mock-id",
						},
					},
				}, nil
			},
			saveFileFn: nil,
			readFileFn: func(path string) ([]byte, error) {
				return nil, errors.New("boom")
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedContains: `{"code": 500, "error": "Could not read file", "reason": "boom"}`,
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

			var saveFileFn func(dst string, file multipart.File) error
			if tc.saveFileFn != nil {
				saveFileFn = tc.saveFileFn
			} else {
				saveFileFn = func(dst string, file multipart.File) error {
					return nil
				}
			}

			var readFileFn func(id string) ([]byte, error)
			if tc.readFileFn != nil {
				readFileFn = tc.readFileFn
			} else {
				readFileFn = func(id string) ([]byte, error) {
					return nil, nil
				}
			}

			storage = &mockStorage{saveFileFn: saveFileFn, readFileFn: readFileFn}

			handler := NewHandler(
				&mockRepo{
					createFunc: tc.createUploadFn,
					getFunc:    tc.getUploadFn,
				},
				storage,
				func() string {
					return "mock-id"
				},
			)
			req := tc.requestSetup(t)
			rr := httptest.NewRecorder()

			r := mux.NewRouter()
			r.HandleFunc("/uploads/{id}/{filename}", handler.Upload)
			r.HandleFunc("/uploads", handler.Uploads)
			r.ServeHTTP(rr, req)

			if tc.expectRaw != nil && *tc.expectRaw == true {
				checkRawResponse(t, rr.Result(), tc.expectedStatus, tc.expectedContains)
			} else {
				checkJSONResponse(t, rr.Result(), tc.expectedStatus, tc.expectedContains)
			}
		})
	}
}

func checkRawResponse(t *testing.T, got *http.Response, expectedStatus int, expected string) {
	defer got.Body.Close()

	body, err := io.ReadAll(got.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %s", err)
	}

	if expectedStatus != got.StatusCode {
		t.Errorf("status mismatch - wanted %d, got %d", expectedStatus, got.StatusCode)
	}

	if expected != string(body) {
		t.Errorf("Body mismatch:\nexpected: %s\ngot: %s", expected, string(body))
	}
}

func checkJSONResponse(t *testing.T, got *http.Response, expectedStatus int, expected string) {
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

func pointerTo[T any](v T) *T {
	return &v
}
