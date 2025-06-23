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

	datasetsRepo := datasets.NewRepository(conn)
	uploadsRepo := uploads.NewRepository(conn)

	fs := &uploads.FileSystemStorage{
		BaseDir: "./files/",
	}

	datasetsSvc := &datasets.Service{
		DatasetsRepo: datasetsRepo,
		UploadsRepo:  uploadsRepo,
		Storage:      fs,
		DB:           conn,
	}

	datasetsHandler := datasets.NewHandler(datasetsSvc)
	mux.HandleFunc("/datasets", datasetsHandler.Datasets)

	uploadsHandler := uploads.NewHandler(uploadsRepo, fs, nil)
	mux.HandleFunc("/uploads", uploadsHandler.Uploads)
	mux.HandleFunc("/uploads/{id}/{filename}", uploadsHandler.Upload)

	loggedMux := loggingMiddleware(mux)
	return loggedMux
}
