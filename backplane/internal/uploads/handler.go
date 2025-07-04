package uploads

import (
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/heldtogether/traintrack/internal"
)

/*
CreateGetter allows an Upload to be created or got from the store.
*/
type CreateGetter interface {
	Create(u *Upload) (*Upload, error)
	Get(id string) (*Upload, error)
}

/*
ReadSaver manages file operations to some storage provider.
*/
type ReadSaver interface {
	SaveFile(dstPath string, file multipart.File) error
	ReadFile(path string) ([]byte, error)
}

/*
UUIDGenerator is a type alias for something that returns unique IDs.
*/
type UUIDGenerator func() string

type Handler struct {
	store   CreateGetter
	storage ReadSaver
	newUUID UUIDGenerator
}

func NewHandler(c CreateGetter, r ReadSaver, uuidGen UUIDGenerator) *Handler {
	if uuidGen == nil {
		uuidGen = func() string {
			return uuid.NewString()
		}
	}
	return &Handler{
		store:   c,
		storage: r,
		newUUID: uuidGen,
	}
}

/*
Uploads routes and handles all requests at the groups of Uploads level.
It should be registered on the router under something sensible, like /uploads.
*/
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

/*
Uploads routes and handles all requests at the individual Upload level.
It should be registered on the router under something sensible.
It expects an `id` and `filename` to be present, like /uploads/{id}/{filename}.
*/
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.Get(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(&internal.Error{
			Code:    http.StatusMethodNotAllowed,
			Message: "Method not allowed",
			Reason:  "",
		})
	}
}

/*
Create accepts a multipart form request consisting of one or more files. It
will store the files in a temporary location on the ReadSaver. We expect
other handlers to later move the files to their forever home.
*/
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

	uploadID := h.newUUID()
	basePath := fmt.Sprintf("tmp/uploads/%s/", uploadID)

	fileRefs := make(map[string]FileRef)
	for artefactName, fileHeaders := range form.File {
		if len(fileHeaders) == 0 {
			continue
		}

		fileHeader := fileHeaders[0]
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

		fileRefs[artefactName] = FileRef{
			Provider: ProviderFileSystem,
			FileName: fileHeader.Filename,
			Path:     basePath,
		}
	}

	if len(fileRefs) == 0 {
		log.Printf("failed to create upload: no files uploaded")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(&internal.Error{
			Code:    http.StatusBadRequest,
			Message: "Failed to create upload",
			Reason:  "no files uploaded",
		})
		return
	}

	upload := &Upload{
		Files: fileRefs,
	}

	upload, err = h.store.Create(upload)
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

/*
Get returns the file `filename` associated with the upload indicated by
`id` in the URL. The file contents is returned, with the correct
Content-Disposition header for details like the filename.
*/
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uploadID := vars["id"]
	filename := vars["filename"]

	upload, err := h.store.Get(uploadID)
	if err != nil || upload == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(&internal.Error{
			Code:    http.StatusNotFound,
			Message: "Upload not found",
			Reason:  err.Error(),
		})
		return
	}

	var filePath string
	var fileName string

	for name, file := range upload.Files {
		if name == filename {
			filePath = filepath.Join(file.Path, file.FileName)
			fileName = file.FileName
			break
		}
	}
	if filePath == "" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(&internal.Error{
			Code:    http.StatusNotFound,
			Message: "File not found",
			Reason:  "unknown file",
		})
		return
	}

	content, err := h.storage.ReadFile(filePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(&internal.Error{
			Code:    http.StatusInternalServerError,
			Message: "Could not read file",
			Reason:  err.Error(),
		})
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}
