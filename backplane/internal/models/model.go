package models

import "encoding/json"

type Model struct {
	ID          string  `json:"id"`
	Name        string  `json:"name" validate:"required"`
	Parent      *string `json:"parent"`
	Version     string  `json:"version" validate:"required"`
	Description string  `json:"description" validate:"required"`

	UploadIds map[string]string `json:"artefacts"`

	DatasetId string `json:"dataset"`

	Config      json.RawMessage `json:"config"`
	Metadata    json.RawMessage `json:"metadata"`
	Environment json.RawMessage `json:"environment"`
	Evaluation  json.RawMessage `json:"evaluation"`
}
