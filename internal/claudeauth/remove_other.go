//go:build !darwin

package claudeauth

// Remove deletes Claude OAuth credentials from the file system.
func Remove() error {
	return removeFile()
}
