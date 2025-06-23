package datasets

type Dataset struct {
	ID          string  `json:"id"`
	Name        string  `json:"name" validate:"required"`
	Parent      *string `json:"parent"`
	Version     string  `json:"version" validate:"required"`
	Description string  `json:"description" validate:"required"`

	UploadIds map[string]string `json:"artefacts"`
}
