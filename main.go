package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
)

var (
	cfg = struct {
		Delete       bool
		Public       bool
		PrintVersion bool
	}{}
	version = "dev"
)

type file struct {
	Filename string
	Size     int64
	MD5      string
}

type filesystemProvider interface {
	WriteFile(path string, content io.ReadSeeker, public bool) error
	ReadFile(path string) (io.ReadCloser, error)
	ListFiles(prefix string) ([]file, error)
	DeleteFile(path string) error
	GetAbsolutePath(path string) (string, error)
}

func main() {
	app := cobra.Command{
		Use:   "s3sync <from> <to>",
		Short: "Sync files from <from> to <to>",
		Run:   execSync,
		PreRun: func(cmd *cobra.Command, args []string) {
			if cfg.PrintVersion {
				fmt.Printf("s3sync %s\n", version)
				os.Exit(0)
			}
		},
	}

	app.Flags().BoolVarP(&cfg.Public, "public", "P", false, "Make files public when syncing to S3")
	app.Flags().BoolVarP(&cfg.Delete, "delete", "d", false, "Delete files on remote not existing on local")
	app.Flags().BoolVarP(&cfg.PrintVersion, "version", "v", false, "Print version and quit")

	app.Execute()
}

func execSync(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		cmd.Usage()
		os.Exit(1)
	}

	local, err := getFSProvider(args[0])
	errExit(err)
	remote, err := getFSProvider(args[1])
	errExit(err)

	localPath, err := local.GetAbsolutePath(args[0])
	errExit(err)
	remotePath, err := remote.GetAbsolutePath(args[1])
	errExit(err)

	localFiles, err := local.ListFiles(localPath)
	errExit(err)
	remoteFiles, err := remote.ListFiles(remotePath)
	errExit(err)

	for i, localFile := range localFiles {
		fmt.Printf("(%d / %d) %s ", i+1, len(localFiles), localFile.Filename)
		needsCopy := true
		for _, remoteFile := range remoteFiles {
			if remoteFile.Filename == localFile.Filename && remoteFile.MD5 == localFile.MD5 {
				needsCopy = false
				break
			}
		}
		if needsCopy {
			l, err := local.ReadFile(path.Join(localPath, localFile.Filename))
			if err != nil {
				fmt.Printf("ERR: %s\n", err)
				continue
			}

			buffer, err := ioutil.ReadAll(l)
			if err != nil {
				fmt.Printf("ERR: %s\n", err)
				continue
			}
			l.Close()

			err = remote.WriteFile(path.Join(remotePath, localFile.Filename), bytes.NewReader(buffer), cfg.Public)
			if err != nil {
				fmt.Printf("ERR: %s\n", err)
				continue
			}

			fmt.Printf("OK\n")
			continue
		}

		fmt.Printf("Skip\n")
	}

	if cfg.Delete {
		for _, remoteFile := range remoteFiles {
			needsDeletion := true
			for _, localFile := range localFiles {
				if localFile.Filename == remoteFile.Filename {
					needsDeletion = false
				}
			}

			if needsDeletion {
				fmt.Printf("delete: %s ", remoteFile.Filename)
				if err := remote.DeleteFile(path.Join(remotePath, remoteFile.Filename)); err != nil {
					fmt.Printf("ERR: %s\n", err)
					continue
				}
				fmt.Printf("OK\n")
			}
		}
	}
}

func errExit(err error) {
	if err != nil {
		fmt.Printf("ERR: %s\n", err)
		os.Exit(1)
	}
}

func getFSProvider(prefix string) (filesystemProvider, error) {
	if strings.HasPrefix(prefix, "s3://") {
		return newS3Provider()
	}
	return newLocalProvider(), nil
}
