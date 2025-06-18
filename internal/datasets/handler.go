package datasets

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/heldtogether/traintrack/internal"
)

type Repo interface {
	List() ([]*Dataset, error)
	Create(d *Dataset) (*Dataset, error)
}

type Handler struct {
	repo Repo
}

func NewHandler(r Repo) *Handler {
	return &Handler{
		repo: r,
	}
}

func (h *Handler) Datasets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.List(w, r)
	case http.MethodPost:
		h.Create(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(&internal.Error{
			Code:    http.StatusMethodNotAllowed,
			Message: "Method not allowed",
			Reason:  "",
		})

	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var d *Dataset
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		log.Printf("failed to decode body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(&internal.Error{
			Code:    http.StatusBadRequest,
			Message: "Failed to create dataset",
			Reason:  fmt.Sprintf("could not parse body: %s", err),
		})
		return
	}

	created, err := h.repo.Create(d)
	if err != nil {
		log.Printf("failed to create dataset: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(&internal.Error{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create dataset",
			Reason:  err.Error(),
		})
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ds, err := h.repo.List()
	if err != nil {
		log.Printf("failed to list datasets: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(&internal.Error{
			Code:    http.StatusInternalServerError,
			Message: "Failed to list datasets",
			Reason:  err.Error(),
		})
		return
	}
	json.NewEncoder(w).Encode(ds)
}
