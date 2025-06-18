package datasets

type Dataset struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Parent      *string `json:"parent"`
	Version     string  `json:"version"`
	Description string  `json:"description"`
}
