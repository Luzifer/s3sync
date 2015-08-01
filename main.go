package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Luzifer/s3sync/logger"
	"github.com/spf13/cobra"
)

var (
	cfg = struct {
		Delete       bool
		Public       bool
		PrintVersion bool
		MaxThreads   int
		logLevel     uint
	}{}
	stdout  *logger.Logger
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
	app.Flags().BoolVar(&cfg.PrintVersion, "version", false, "Print version and quit")
	app.Flags().IntVar(&cfg.MaxThreads, "max-threads", 10, "Use max N parallel threads for file sync")
	app.Flags().UintVar(&cfg.logLevel, "loglevel", 2, "Amount of log output (0 = Error only, 3 = Debug)")

	app.ParseFlags(os.Args[1:])

	stdout = logger.New(logger.LogLevel(cfg.logLevel))

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

	syncChannel := make(chan bool, cfg.MaxThreads)
	for i, localFile := range localFiles {
		syncChannel <- true
		go func(i int, localFile file) {
			defer func() { <-syncChannel }()

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
					stdout.ErrorF("(%d / %d) %s ERR: %s\n", i+1, len(localFiles), localFile.Filename, err)
					return
				}

				buffer, err := ioutil.ReadAll(l)
				if err != nil {
					stdout.ErrorF("(%d / %d) %s ERR: %s\n", i+1, len(localFiles), localFile.Filename, err)
					return
				}
				l.Close()

				err = remote.WriteFile(path.Join(remotePath, localFile.Filename), bytes.NewReader(buffer), cfg.Public)
				if err != nil {
					stdout.ErrorF("(%d / %d) %s ERR: %s\n", i+1, len(localFiles), localFile.Filename, err)
					return
				}

				stdout.InfoF("(%d / %d) %s OK\n", i+1, len(localFiles), localFile.Filename)
				return
			}

			stdout.DebugF("(%d / %d) %s Skip\n", i+1, len(localFiles), localFile.Filename)
		}(i, localFile)
	}

	if cfg.Delete {
		for _, remoteFile := range remoteFiles {
			syncChannel <- true
			go func(remoteFile file) {
				defer func() { <-syncChannel }()

				needsDeletion := true
				for _, localFile := range localFiles {
					if localFile.Filename == remoteFile.Filename {
						needsDeletion = false
					}
				}

				if needsDeletion {
					if err := remote.DeleteFile(path.Join(remotePath, remoteFile.Filename)); err != nil {
						stdout.ErrorF("delete: %s ERR: %s\n", remoteFile.Filename, err)
						return
					}
					stdout.InfoF("delete: %s OK\n", remoteFile.Filename)
				}
			}(remoteFile)
		}
	}

	for {
		if len(syncChannel) == 0 {
			break
		}
		<-time.After(time.Second)
	}
}

func errExit(err error) {
	if err != nil {
		stdout.ErrorF("ERR: %s\n", err)
		os.Exit(1)
	}
}

func getFSProvider(prefix string) (filesystemProvider, error) {
	if strings.HasPrefix(prefix, "s3://") {
		return newS3Provider()
	}
	return newLocalProvider(), nil
}
