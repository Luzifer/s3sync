package fsprovider

import (
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	channelBufferSizeHuge  = 10000
	channelBufferSizeSmall = 10
	doneTickerInterval     = 500 * time.Millisecond
	maxKeysPerPage         = 1000
)

type (
	// S3 implements the Provider for S3 / MinIO access
	S3 struct {
		conn            *s3.S3
		requestedPrefix string
	}
)

// NewS3 creates a new S3 / MinIO file provider
func NewS3(endpoint string) (*S3, error) {
	var cfgs []*aws.Config

	if endpoint != "" {
		cfgs = append(cfgs, &aws.Config{
			Endpoint:         &endpoint,
			S3ForcePathStyle: aws.Bool(true),
		})
	}

	sess := session.Must(session.NewSession(cfgs...))
	return &S3{
		conn: s3.New(sess),
	}, nil
}

// DeleteFile deletes an object from the bucket
func (s *S3) DeleteFile(path string) error {
	bucket, path, err := s.getBucketPath(path)
	if err != nil {
		return errors.Wrap(err, "getting bucket path")
	}

	_, err = s.conn.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	})

	return errors.Wrap(err, "deleting object")
}

// GetAbsolutePath converts the given path into an absolute path
func (*S3) GetAbsolutePath(path string) (string, error) {
	return path, nil
}

// ListFiles retrieves the metadata of all objects with given prefix
func (s *S3) ListFiles(prefix string) ([]File, error) {
	out := []File{}

	bucket, path, err := s.getBucketPath(prefix)
	if err != nil {
		return out, errors.Wrap(err, "getting bucket path")
	}

	processedPrefixes := []string{}

	prefixChan := make(chan *string, channelBufferSizeHuge)
	outputChan := make(chan File, channelBufferSizeHuge)
	errChan := make(chan error, channelBufferSizeSmall)
	syncChan := make(chan bool, channelBufferSizeSmall)
	doneTimer := time.NewTicker(doneTickerInterval)

	prefixChan <- aws.String(path)

	for {
		select {
		case prefix := <-prefixChan:
			if len(syncChan) == channelBufferSizeSmall {
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
			logrus.Debugf("scanning prefixes (%d working, %d left)...", len(syncChan), len(prefixChan))
			if len(prefixChan) == 0 && len(syncChan) == 0 {
				return out, nil
			}
		}
	}
}

// ReadFile retrieves the object body for reading
func (s *S3) ReadFile(path string) (io.ReadCloser, error) {
	bucket, path, err := s.getBucketPath(path)
	if err != nil {
		return nil, errors.Wrap(err, "getting bucket path")
	}

	o, err := s.conn.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, errors.Wrap(err, "getting object")
	}

	return o.Body, nil
}

// WriteFile copies the content into an S3 object
//
//revive:disable-next-line:flag-parameter // That's not a control coupling but a config flag
func (s *S3) WriteFile(path string, content io.Reader, public bool) error {
	bucket, path, err := s.getBucketPath(path)
	if err != nil {
		return errors.Wrap(err, "getting bucket path")
	}

	ext := filepath.Ext(path)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	params := &s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(path),
		Body:        content,
		ContentType: aws.String(mimeType),
	}
	if public {
		params.ACL = aws.String("public-read")
	}

	_, err = s3manager.NewUploaderWithClient(s.conn).Upload(params)
	return errors.Wrap(err, "uploading file")
}

func (s *S3) getBucketPath(prefix string) (bucket string, path string, err error) {
	rex := regexp.MustCompile(`^s3://?([^/]+)/(.*)$`)
	matches := rex.FindStringSubmatch(prefix)
	if matches == nil {
		return "", "", errors.New("prefix did not match requirements")
	}

	bucket = matches[1]
	path = strings.ReplaceAll(matches[2], string(os.PathSeparator), "/")
	s.requestedPrefix = path

	return bucket, path, nil
}

func (s *S3) readS3FileList(bucket string, path *string, outputChan chan File, prefixChan chan *string, errorChan chan error, syncChan chan bool) {
	defer func() { <-syncChan }()
	in := &s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    path,
		MaxKeys:   aws.Int64(maxKeysPerPage),
		Delimiter: aws.String("/"),
	}
	for {
		o, err := s.conn.ListObjects(in)
		if err != nil {
			errorChan <- errors.Wrap(err, "listing objects")
			return
		}

		for _, v := range o.Contents {
			outputChan <- File{
				Filename:     strings.Replace(*v.Key, s.requestedPrefix, "", 1),
				LastModified: *v.LastModified,
				Size:         *v.Size,
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
