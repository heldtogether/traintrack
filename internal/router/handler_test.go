package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetup(t *testing.T) {
	router := Setup(nil)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", w.Code)
	}
}
