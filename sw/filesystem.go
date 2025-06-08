package sw

import (
	"os"
)

// OSFileSystem is a concrete implementation of FileSystem using the OS
type OSFileSystem struct{}

// NewOSFileSystem creates a new OS-based file system
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

// WriteFile writes data to a file
func (fs *OSFileSystem) WriteFile(path string, data []byte, perm int) error {
	return os.WriteFile(path, data, os.FileMode(perm))
}

// Exists checks if a file exists
func (fs *OSFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// MkdirAll creates directories recursively
func (fs *OSFileSystem) MkdirAll(path string, perm int) error {
	return os.MkdirAll(path, os.FileMode(perm))
}
