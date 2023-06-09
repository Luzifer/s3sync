// Package fsprovider contains implementations for filesystem access
// to read files from / write files to
package fsprovider

import (
	"io"
	"time"
)

type (
	// File contains metadata about the file to be copied
	File struct {
		Filename     string
		LastModified time.Time
		Size         int64
	}

	// Provider describes the implementation of a fsprovider
	Provider interface {
		WriteFile(path string, content io.Reader, public bool) error
		ReadFile(path string) (io.ReadCloser, error)
		ListFiles(prefix string) ([]File, error)
		DeleteFile(path string) error
		GetAbsolutePath(path string) (string, error)
	}
)
