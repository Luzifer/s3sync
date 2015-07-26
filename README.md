# Luzifer / s3sync

[![License: Apache v2.0](https://badge.luzifer.io/v1/badge?color=5d79b5&title=license&text=Apache+v2.0)](http://www.apache.org/licenses/LICENSE-2.0) [![Gobuild Download](https://badge.luzifer.io/v1/badge?color=5d79b5&title=Download&text=on+GoBuilder.me)](https://gobuilder.me/github.com/Luzifer/s3sync)

`s3sync` is a small utility to sync local directories from/to Amazon S3 without installing any dependencies. Just put the binary somewhere into your path and set three ENV variables and your're ready to sync.

## Features

- Static binary, no dependencies required
- Sync files only if required (judged by MD5 checksum)
- Optionally delete files at target
- Optionally make files public on sync (only if file needs sync)
- Sync local-to-s3, s3-to-local, local-to-local or s3-to-s3

## Usage

1. Set `AWS_ACCESS_KEY`, `AWS_SECRET_ACCESS_KEY` and `AWS_REGION`
2. Execute your sync

```bash
# s3sync --help
Sync files from <from> to <to>

Usage:
  s3sync <from> <to> [flags]

Flags:
  -d, --delete=false: Delete files on remote not existing on local
  -h, --help=false: help for s3sync
      --max-threads=10: Use max N parallel threads for file sync
  -P, --public=false: Make files public when syncing to S3
  -v, --version=false: Print version and quit

# s3sync -d 1/ s3://knut-test-s3sync/
(1 / 3) 05/11/pwd_luzifer_io.png OK
(2 / 3) 07/26/bkm.png OK
(3 / 3) 07/26/luzifer_io.png OK

# s3sync -d 1/ s3://knut-test-s3sync/
(1 / 3) 05/11/pwd_luzifer_io.png Skip
(2 / 3) 07/26/bkm.png Skip
(3 / 3) 07/26/luzifer_io.png Skip

# rm -rf 1/05
# s3sync -d s3://knut-test-s3sync/ 1/
(1 / 3) 05/11/pwd_luzifer_io.png OK
(2 / 3) 07/26/bkm.png Skip
(3 / 3) 07/26/luzifer_io.png Skip
```
