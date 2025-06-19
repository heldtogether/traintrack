package uploads

import (
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/google/uuid"
	"github.com/heldtogether/traintrack/internal"
)

type Repo interface {
	Create(u *Upload) (*Upload, error)
}

type Storage interface {
	SaveFile(dstPath string, file multipart.File) error
}

type UUIDGenerator func() string

type Handler struct {
	repo    Repo
	storage Storage
	newUUID UUIDGenerator
}

func NewHandler(r Repo, s Storage, uuidGen UUIDGenerator) *Handler {
	if uuidGen == nil {
		uuidGen = func() string {
			return uuid.NewString()
		}
	}
	return &Handler{
		repo:    r,
		storage: s,
		newUUID: uuidGen,
	}
}

func (h *Handler) Uploads(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
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
	err := r.ParseMultipartForm(32 << 20) // 32MB chunks
	if err != nil {
		log.Printf("failed to create upload: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(&internal.Error{
			Code:    http.StatusBadRequest,
			Message: "Failed to create upload",
			Reason:  err.Error(),
		})
		return
	}

	form := r.MultipartForm
	files := form.File["files"]
	if len(files) == 0 {
		log.Printf("failed to create upload: no files uploaded")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(&internal.Error{
			Code:    http.StatusBadRequest,
			Message: "Failed to create upload",
			Reason:  "no files uploaded",
		})
		return
	}

	uploadID := h.newUUID()
	basePath := fmt.Sprintf("tmp/uploads/%s/", uploadID)

	var fileRefs []FileRef
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			// this branch isn't tested as we don't expect
			// multipart files to not open in regular usage
			log.Printf("failed to create upload: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(&internal.Error{
				Code:    http.StatusInternalServerError,
				Message: "Failed to create upload",
				Reason:  err.Error(),
			})
			return
		}
		defer file.Close()

		dst := basePath + fileHeader.Filename
		err = h.storage.SaveFile(dst, file)
		if err != nil {
			log.Printf("failed to create upload: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(&internal.Error{
				Code:    http.StatusInternalServerError,
				Message: "Failed to create upload",
				Reason:  err.Error(),
			})
			return
		}

		fileRefs = append(fileRefs, FileRef{
			Provider: ProviderFileSystem,
			FileName: fileHeader.Filename,
			Path:     basePath,
		})
	}

	upload := &Upload{
		Files: fileRefs,
	}

	upload, err = h.repo.Create(upload)
	if err != nil {
		log.Printf("failed to create upload: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(&internal.Error{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create upload",
			Reason:  err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(upload)
}
