package janet

// BlankUIProvider provides a no-op UI service
type BlankUIProvider struct{}

// GetURL returns an empty string, signifying that the UI is disabled
func (p *BlankUIProvider) GetURL(path string) (string, error) {
	return "", nil
}
