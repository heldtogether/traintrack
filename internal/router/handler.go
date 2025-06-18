package router

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func Setup() http.Handler {
	mux := mux.NewRouter()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	loggedMux := loggingMiddleware(mux)
	return loggedMux
}
