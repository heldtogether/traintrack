package datasets

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/heldtogether/traintrack/internal"
)

/*
Creator allows a Dataset to be created.
*/
type Creator interface {
	Create(ctx context.Context, d *Dataset) (*Dataset, error)
}

/*
Lister allows Datasets to be listed.
*/
type Lister interface {
	List() ([]*Dataset, error)
}

type Handler struct {
	c Creator
	l Lister

	validator *validator.Validate
	trans     ut.Translator
}

func NewHandler(c Creator, l Lister) *Handler {
	validator := validator.New(validator.WithRequiredStructEnabled())
	validator.RegisterTagNameFunc(func(fld reflect.StructField) string {
		tag := fld.Tag.Get("json")
		if tag == "-" {
			return ""
		}
		name := strings.SplitN(tag, ",", 2)[0]
		return name
	})

	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)
	trans, _ := uni.GetTranslator("en")
	en_translations.RegisterDefaultTranslations(validator, trans)

	return &Handler{
		c:         c,
		l:         l,
		validator: validator,
		trans:     trans,
	}
}

/*
Datasets routes and handles all requests for Datasets. It should be
registered on the router under something sensible, like /datasets.
*/
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

	if err := h.validator.Struct(d); err != nil {
		details := map[string]string{}
		for _, e := range err.(validator.ValidationErrors) {
			details[e.Field()] = e.Translate(h.trans)
		}
		log.Printf("failed to validate input: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(&internal.Error{
			Code:    http.StatusBadRequest,
			Message: "Failed to create dataset",
			Reason:  "bad input",
			Details: details,
		})
		return
	}

	created, err := h.c.Create(r.Context(), d)
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
	ds, err := h.l.List()
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
