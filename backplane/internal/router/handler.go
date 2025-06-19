package router

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/heldtogether/traintrack/internal/datasets"
	"github.com/heldtogether/traintrack/internal/uploads"
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

	fs := &uploads.FileSystemStorage{
		BaseDir: "./files/",
	}

	uploadsHandler := uploads.NewHandler(uploads.NewRepository(conn), fs, nil)
	mux.HandleFunc("/uploads", uploadsHandler.Uploads)

	loggedMux := loggingMiddleware(mux)
	return loggedMux
}
