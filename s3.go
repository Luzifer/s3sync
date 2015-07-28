package main

import (
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

type s3Provider struct {
	conn *s3.S3
}

func newS3Provider() (*s3Provider, error) {
	return &s3Provider{
		conn: s3.New(&aws.Config{}),
	}, nil
}

func (s *s3Provider) getBucketPath(prefix string) (bucket string, path string, err error) {
	rex := regexp.MustCompile(`^s3://?([^/]+)/(.*)$`)
	matches := rex.FindStringSubmatch(prefix)
	if len(matches) != 3 {
		err = fmt.Errorf("prefix did not match requirements")
		return
	}

	bucket = matches[1]
	path = matches[2]

	return
}

func (s *s3Provider) ListFiles(prefix string) ([]file, error) {
	out := []file{}

	bucket, path, err := s.getBucketPath(prefix)
	if err != nil {
		return out, err
	}

	processedPrefixes := []string{}

	prefixChan := make(chan *string, 10000)
	outputChan := make(chan file, 10000)
	errChan := make(chan error, 10)
	syncChan := make(chan bool, 10)
	doneTimer := time.NewTicker(500 * time.Millisecond)

	prefixChan <- aws.String(path)

	for {
		select {
		case prefix := <-prefixChan:
			if len(syncChan) == 10 {
				prefixChan <- prefix
			} else {
				found := false
				for _, v := range processedPrefixes {
					if v == *prefix {
						found = true
					}
				}
				if !found {
					syncChan <- true
					go s.readS3FileList(bucket, prefix, outputChan, prefixChan, errChan, syncChan)
					processedPrefixes = append(processedPrefixes, *prefix)
				}
			}
		case o := <-outputChan:
			out = append(out, o)
		case err := <-errChan:
			return out, err
		case <-doneTimer.C:
			stdout.DebugF("Scanning prefixes (%d working, %d left)...\r", len(syncChan), len(prefixChan))
			if len(prefixChan) == 0 && len(syncChan) == 0 {
				fmt.Printf("\n")
				return out, nil
			}
		}
	}
}

func (s *s3Provider) readS3FileList(bucket string, path *string, outputChan chan file, prefixChan chan *string, errorChan chan error, syncChan chan bool) {
	defer func() { <-syncChan }()
	in := &s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    path,
		MaxKeys:   aws.Long(1000),
		Delimiter: aws.String("/"),
	}
	for {
		o, err := s.conn.ListObjects(in)
		if err != nil {
			errorChan <- err
			return
		}

		for _, v := range o.Contents {
			outputChan <- file{
				Filename: *v.Key,
				Size:     *v.Size,
				MD5:      strings.Trim(*v.ETag, "\""), // Wat?
			}
		}

		if len(o.CommonPrefixes) > 0 {
			for _, cp := range o.CommonPrefixes {
				prefixChan <- cp.Prefix
			}
		}

		if !*o.IsTruncated {
			break
		}
		in.Marker = o.NextMarker
	}
}

func (s *s3Provider) WriteFile(path string, content io.ReadSeeker, public bool) error {
	bucket, path, err := s.getBucketPath(path)
	if err != nil {
		return err
	}

	ext := filepath.Ext(path)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	params := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(path),
		Body:        content,
		ContentType: aws.String(mimeType),
	}
	if public {
		params.ACL = aws.String("public-read")
	}
	_, err = s.conn.PutObject(params)

	return err
}

func (s *s3Provider) ReadFile(path string) (io.ReadCloser, error) {
	bucket, path, err := s.getBucketPath(path)
	if err != nil {
		return nil, err
	}

	o, err := s.conn.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		return nil, err
	}

	return o.Body, nil
}

func (s *s3Provider) DeleteFile(path string) error {
	bucket, path, err := s.getBucketPath(path)
	if err != nil {
		return err
	}

	_, err = s.conn.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	})

	return err
}

func (s *s3Provider) GetAbsolutePath(path string) (string, error) {
	return path, nil
}
