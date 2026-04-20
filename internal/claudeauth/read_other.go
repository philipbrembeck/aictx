//go:build !darwin

package claudeauth

// Read returns the current Claude OAuth credentials from the file system.
func Read() (string, error) {
	return readFromFile()
}
