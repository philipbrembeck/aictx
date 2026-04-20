//go:build !darwin

package claudeauth

// Write writes Claude OAuth credentials to the file system.
func Write(credentials string) error {
	return writeToFile(credentials)
}
