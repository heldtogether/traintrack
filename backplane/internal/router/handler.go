package router

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/heldtogether/traintrack/internal/datasets"
	"github.com/heldtogether/traintrack/internal/models"
	"github.com/heldtogether/traintrack/internal/uploads"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Setup(conn *pgxpool.Pool) http.Handler {
	mux := mux.NewRouter()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	datasetsStore := datasets.NewStore(conn)
	uploadsStore := uploads.NewStore(conn)
	modelsStore := models.NewStore(conn)

	fs := &uploads.FileSystemStore{
		BaseDir: "./files/",
	}

	datasetsCreator := datasets.NewCreator(
		datasetsStore,
		uploadsStore,
		fs,
		conn,
	)

	modelsCreator := models.NewCreator(
		modelsStore,
		uploadsStore,
		fs,
		conn,
	)

	datasetsHandler := datasets.NewHandler(datasetsCreator, datasetsStore)
	mux.HandleFunc("/datasets", datasetsHandler.Datasets)

	uploadsHandler := uploads.NewHandler(uploadsStore, fs, nil)
	mux.HandleFunc("/uploads", uploadsHandler.Uploads)
	mux.HandleFunc("/uploads/{id}/{filename}", uploadsHandler.Upload)

	modelsHandler := models.NewHandler(modelsCreator, modelsStore)
	mux.HandleFunc("/models", modelsHandler.Models)

	loggedMux := loggingMiddleware(mux)
	return loggedMux
}
