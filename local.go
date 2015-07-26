package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type localProvider struct{}

func newLocalProvider() *localProvider {
	return &localProvider{}
}

func (l *localProvider) WriteFile(path string, content io.ReadSeeker, public bool) error {
	os.MkdirAll(filepath.Dir(path), 0755)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, content); err != nil {
		return err
	}
	return f.Close()
}

func (l *localProvider) ReadFile(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

func (l *localProvider) ListFiles(prefix string) ([]file, error) {
	out := []file{}

	err := filepath.Walk(prefix, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !f.IsDir() {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			out = append(out, file{
				Filename: strings.TrimLeft(strings.Replace(path, prefix, "", 1), "/"),
				Size:     f.Size(),
				MD5:      fmt.Sprintf("%x", md5.Sum(content)),
			})
		}
		return nil
	})

	return out, err
}

func (l *localProvider) DeleteFile(path string) error {
	return os.Remove(path)
}

func (l *localProvider) GetAbsolutePath(path string) (string, error) {
	return filepath.Abs(path)
}
