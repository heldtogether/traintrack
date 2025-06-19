package uploads

type Upload struct {
	ID    string    `json:"id"`
	Files []FileRef `json:"files"`
}

type Provider string

const (
	ProviderUnknown    Provider = "unknown"
	ProviderFileSystem          = "filesystem"
)

type FileRef struct {
	Provider Provider `json:"provider"`
	FileName string   `json:"filename"`
	Path     string   `json:"path"`
}
