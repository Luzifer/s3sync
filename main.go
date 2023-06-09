package main

import (
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/Luzifer/rconfig/v2"
	"github.com/Luzifer/s3sync/pkg/fsprovider"
)

var (
	cfg = struct {
		Delete         bool   `flag:"delete,d" default:"false" description:"Delete files on remote not existing on local"`
		Endpoint       string `flag:"endpoint" default:"" description:"Switch S3 endpoint (i.e. for MinIO compatibility)"`
		LogLevel       string `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		MaxThreads     int    `flag:"max-threads" default:"10" description:"Use max N parallel threads for file sync"`
		Public         bool   `flag:"public,P" default:"false" description:"Make files public when syncing to S3"`
		VersionAndExit bool   `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	version = "dev"
)

func getFSProvider(prefix string) (fsprovider.Provider, error) {
	if strings.HasPrefix(prefix, "s3://") {
		p, err := fsprovider.NewS3(cfg.Endpoint)
		return p, errors.Wrap(err, "getting s3 provider")
	}
	return fsprovider.NewLocal(), nil
}

func initApp() error {
	rconfig.AutoEnv(true)
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		return errors.Wrap(err, "parsing cli options")
	}

	l, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return errors.Wrap(err, "parsing log-level")
	}
	logrus.SetLevel(l)

	return nil
}

func main() {
	var err error
	if err = initApp(); err != nil {
		logrus.WithError(err).Fatal("initializing app")
	}

	if cfg.VersionAndExit {
		logrus.WithField("version", version).Info("s3sync")
		os.Exit(0)
	}

	if err = runSync(rconfig.Args()[1:]); err != nil {
		logrus.WithError(err).Fatal("running sync")
	}
}

//nolint:funlen,gocognit,gocyclo // Should be kept as single unit
func runSync(args []string) error {
	//nolint:gomnd // Simple count of parameters, makes no sense to export
	if len(args) != 2 {
		return errors.New("missing required arguments: s3sync <from> <to>")
	}

	local, err := getFSProvider(args[0])
	if err != nil {
		return errors.Wrap(err, "getting local provider")
	}
	remote, err := getFSProvider(args[1])
	if err != nil {
		return errors.Wrap(err, "getting remote provider")
	}

	localPath, err := local.GetAbsolutePath(args[0])
	if err != nil {
		return errors.Wrap(err, "getting local path")
	}
	remotePath, err := remote.GetAbsolutePath(args[1])
	if err != nil {
		return errors.Wrap(err, "getting remote path")
	}

	localFiles, err := local.ListFiles(localPath)
	if err != nil {
		return errors.Wrap(err, "listing local files")
	}
	remoteFiles, err := remote.ListFiles(remotePath)
	if err != nil {
		return errors.Wrap(err, "listing remote files")
	}

	var (
		nErr        int
		syncChannel = make(chan bool, cfg.MaxThreads)
	)

	for i, localFile := range localFiles {
		syncChannel <- true
		go func(i int, localFile fsprovider.File) {
			defer func() { <-syncChannel }()

			var (
				logger      = logrus.WithField("filename", localFile.Filename)
				debugLogger = logger.WithField("tx_reason", "missing")

				needsCopy   bool
				remoteFound bool
			)

			for _, remoteFile := range remoteFiles {
				if remoteFile.Filename != localFile.Filename {
					// Different file, do not compare
					continue
				}

				// We found a match, lets check whether tx is required
				remoteFound = true

				switch {
				case remoteFile.Size != localFile.Size:
					debugLogger = debugLogger.WithField("tx_reason", "size-mismatch").WithField("ls", localFile.Size).WithField("rs", remoteFile.Size)
					needsCopy = true

				case localFile.LastModified.After(remoteFile.LastModified):
					debugLogger = debugLogger.WithField("tx_reason", "local-newer")
					needsCopy = true

				default:
					// No reason to update
					needsCopy = false
				}

				break
			}

			if remoteFound && !needsCopy {
				logger.Debug("skipped transfer")
				return
			}

			debugLogger.Debug("starting transfer")

			l, err := local.ReadFile(path.Join(localPath, localFile.Filename))
			if err != nil {
				logger.WithError(err).Error("reading local file")
				nErr++
				return
			}
			defer func() {
				if err := l.Close(); err != nil {
					logger.WithError(err).Error("closing local file")
				}
			}()

			err = remote.WriteFile(path.Join(remotePath, localFile.Filename), l, cfg.Public)
			if err != nil {
				logger.WithError(err).Error("writing remote file")
				nErr++
				return
			}

			logger.Info("transferred file")
		}(i, localFile)
	}

	if cfg.Delete {
		for _, remoteFile := range remoteFiles {
			syncChannel <- true
			go func(remoteFile fsprovider.File) {
				defer func() { <-syncChannel }()

				needsDeletion := true
				for _, localFile := range localFiles {
					if localFile.Filename == remoteFile.Filename {
						needsDeletion = false
					}
				}

				if needsDeletion {
					if err := remote.DeleteFile(path.Join(remotePath, remoteFile.Filename)); err != nil {
						logrus.WithField("filename", remoteFile.Filename).WithError(err).Error("deleting remote file")
						nErr++
						return
					}
					logrus.WithField("filename", remoteFile.Filename).Info("deleted remote file")
				}
			}(remoteFile)
		}
	}

	for len(syncChannel) > 0 {
		<-time.After(time.Second)
	}

	return nil
}
