package janet

// Provider represents a UI provider interface
type Provider interface {
	GetURL(path string) (string, error)
}
