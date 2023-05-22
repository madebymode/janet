package ui

// A Provider provides a UI service that can be
// attached to janet.
type Provider interface {
	GetURL(URI string) (string, error)
	Listen() error
}
