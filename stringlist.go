package janet

// StringList represents a set of strings
type StringList map[string]struct{}

// Contains checks if a string is in the list
func (s StringList) Contains(str string) bool {
	_, exists := s[str]
	return exists
}
