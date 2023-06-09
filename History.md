# 2.0.0 / 2023-06-09

**⚠ Breaking Changes:**
- MD5 checksums are no longer used to check for file changes
- Logging to stdout was dropped and log-format changed

**ℹ Changelog:**

  * Breaking: Refactor, update deps, add MinIO support
  * Replace deprecated vendoring with Go modules support

# 1.3.0 / 2017-11-13

  * Update dependencies, switch to dep for vendoring
  * Update meta files

# 1.2.6 / 2017-05-29

  * Migrate and update Godeps

# 1.2.5 / 2017-05-29

  * Noop: Enable Github releases


1.2.4 / 2016-03-23
==================

  * Fix: Allow sync of subpaths from S3

1.2.3 / 2016-02-29
==================

  * Fix: Synced files from Windows did not have subdirs

1.2.2 / 2015-08-02
==================

  * Fix: Updated aws-sdk and solved issues with latest version

1.2.1 / 2015-08-01
==================

  * Fix: Do not exit before every sync is done

1.2.0 / 2015-07-28
==================

  * Added levels to logging to silence output if required

1.1.1 / 2015-07-26
==================

  * Fix: Remove `-v` shorthand as it lead to confusion with "verbose" flags
  * Fix: Made channels bigger for S3 processing

1.1.0 / 2015-07-26
==================

  * Fetch s3 file list in parallel
  * Do file sync with more than 1 thread in parallel

1.0.1 / 2015-07-26
==================

  * Fix: List logic was not able to list more than 1000 files
  * Fix: Move version info to flag instead of command

1.0.0 / 2015-07-26
==================

 * Initial version
