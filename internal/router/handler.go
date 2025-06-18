package router

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/heldtogether/traintrack/internal/datasets"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Setup(conn *pgxpool.Pool) http.Handler {
	mux := mux.NewRouter()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	datasetsHandler := datasets.NewHandler(datasets.NewRepository(conn))
	mux.HandleFunc("/datasets", datasetsHandler.Datasets)

	loggedMux := loggingMiddleware(mux)
	return loggedMux
}
