package uploads

type Upload struct {
	ID        string             `json:"id"`
	Files     map[string]FileRef `json:"files"`
	DatasetID *string             `json:"dataset_id,omitempty"`
	ModelID   *string             `json:"model_id,omitempty"`
}

type Provider string

const (
	ProviderUnknown    Provider = "unknown"
	ProviderFileSystem Provider = "filesystem"
)

type FileRef struct {
	Provider Provider `json:"provider"`
	FileName string   `json:"filename"`
	Path     string   `json:"path"`
}
