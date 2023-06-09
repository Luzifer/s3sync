package fsprovider

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

const directoryCreatePerms = 0o750

type (
	// Local implements the Provider for filesystem access
	Local struct{}
)

// NewLocal creates a new Local file provider
func NewLocal() *Local {
	return &Local{}
}

// DeleteFile deletes a local file
func (*Local) DeleteFile(path string) error {
	return errors.Wrap(os.Remove(path), "removing file")
}

// GetAbsolutePath converts the given path into an absolute path
func (*Local) GetAbsolutePath(path string) (string, error) {
	fp, err := filepath.Abs(path)
	return fp, errors.Wrap(err, "getting absolute filepath")
}

// ListFiles retrieves the metadata of all files with given prefix
func (*Local) ListFiles(prefix string) ([]File, error) {
	out := []File{}

	err := filepath.Walk(prefix, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if f.IsDir() {
			return nil
		}

		linkTarget, err := filepath.EvalSymlinks(path)
		if err != nil {
			return errors.Wrap(err, "evaluating symlinks")
		}

		if path != linkTarget {
			if f, err = os.Stat(linkTarget); err != nil {
				return errors.Wrap(err, "getting file-stat of link target")
			}
		}

		out = append(out, File{
			Filename:     strings.TrimLeft(strings.Replace(path, prefix, "", 1), string(os.PathSeparator)),
			LastModified: f.ModTime(),
			Size:         f.Size(),
		})

		return nil
	})

	return out, errors.Wrap(err, "walking prefix")
}

// ReadFile opens the local file for reading
func (*Local) ReadFile(path string) (io.ReadCloser, error) {
	f, err := os.Open(path) //#nosec:G304 // The purpose is to read a dynamic file
	return f, errors.Wrap(err, "opening file")
}

// WriteFile takes the content of the file and writes it to the local
// filesystem
func (*Local) WriteFile(path string, content io.Reader, _ bool) error {
	if err := os.MkdirAll(filepath.Dir(path), directoryCreatePerms); err != nil {
		return errors.Wrap(err, "creating file path")
	}

	f, err := os.Create(path) //#nosec:G304 // The purpose is to create a dynamic file
	if err != nil {
		return errors.Wrap(err, "creating file")
	}
	if _, err := io.Copy(f, content); err != nil {
		return errors.Wrap(err, "copying content")
	}
	return errors.Wrap(f.Close(), "closing file")
}
