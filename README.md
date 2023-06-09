# Luzifer / s3sync

![](https://badges.fyi/github/license/Luzifer/s3sync)
![](https://badges.fyi/github/downloads/Luzifer/s3sync)

`s3sync` is a small utility to sync local directories from/to Amazon S3 without installing any dependencies. Just put the binary somewhere into your path and set three ENV variables and your're ready to sync.

## Features

- Static binary, no dependencies required
- Sync files only if required (judged by file size & modify-date)
- Using multiple threads to upload the transfer is quite fast
- Optionally delete files at target
- Optionally make files public on sync (only if file needs sync)
- Sync local-to-s3, s3-to-local, local-to-local or s3-to-s3

## Usage

1. Set `AWS_ACCESS_KEY`, `AWS_SECRET_ACCESS_KEY` and `AWS_REGION`
2. Execute your sync

```bash
# s3sync --help
Usage of s3sync:
  -d, --delete             Delete files on remote not existing on local
      --endpoint string    Switch S3 endpoint (i.e. for MinIO compatibility)
      --log-level string   Log level (debug, info, warn, error, fatal) (default "info")
      --max-threads int    Use max N parallel threads for file sync (default 10)
  -P, --public             Make files public when syncing to S3
      --version            Prints current version and exits


# echo "ello" >test.txt
# echo "foobar" >test2.txt

# s3sync --delete ./ s3://test/
time="2023-06-09T17:15:09+02:00" level=info msg="transferred file" filename=test2.txt
time="2023-06-09T17:15:09+02:00" level=info msg="transferred file" filename=test.txt

# s3sync --delete ./ s3://test/

# touch test.txt
# rm test2.txt

# s3sync --delete ./ s3://test/
time="2023-06-09T17:15:29+02:00" level=info msg="deleted remote file" filename=test2.txt
time="2023-06-09T17:15:29+02:00" level=info msg="transferred file" filename=test.txt
```
