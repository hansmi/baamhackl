# Baamhackl

[![Latest release](https://img.shields.io/github/v/release/hansmi/baamhackl)][releases]
[![Release workflow](https://github.com/hansmi/baamhackl/actions/workflows/release.yaml/badge.svg)](https://github.com/hansmi/baamhackl/actions/workflows/release.yaml)
[![CI workflow](https://github.com/hansmi/baamhackl/actions/workflows/ci.yaml/badge.svg)](https://github.com/hansmi/baamhackl/actions/workflows/ci.yaml)
[![Go reference](https://pkg.go.dev/badge/github.com/hansmi/baamhackl.svg)](https://pkg.go.dev/github.com/hansmi/baamhackl)

Execute commands for new and changed files in directories. Builds on
[Facebook's Watchman][watchman], a file watching service. The author uses
Baamhackl to pass PDF files produced by a network-connected scanner through
[OCRmyPDF](https://github.com/jbarlow83/OCRmyPDF/) and other tools.

["Baamhackl"](https://bar.wikipedia.org/wiki/Baamhackl) is
[Bavarian](https://en.wikipedia.org/wiki/Bavarian_language) for a woodpecker.


## Usage

A YAML-formatted configuration file is required to define observed directories
and handler commands. Example:

```yaml
handlers:
  - name: scanned
    path: /srv/shared/scanned
    command: ["/bin/bash", "-c", "echo ${BAAMHACKL_INPUT}"]
```

Every time a file is created in or moved into the `/srv/shared/scanned`
directory the given handler command is launched. File modifications while the
command is running are considered a handler failure triggering a retry.

A log of per-file actions taken is recorded in the journal directory located at
`_/journal` relative to the observed directory, i.e.
`/srv/shared/scanned/_/journal` in the example above. After the handler command
succeeded the originally changed file is moved to the `_/success` directory. In
case of exhausting all retries a file for which the command fails consistently
is moved to the `_/failure` directory.

Command logs, successful and failed files are cleaned up periodically.

To use Baamhackl a Watchman server must already be running and accessible (e.g.
launched via systemd or another service manager). For debugging purposes an
instance can be launched in the foreground:

```shell
watchman --foreground --log-level=1 --logfile=/dev/stderr
```

The number of handler commands to run concurrently can be configured with
`baamhackl watch -slots=N`.

The `baamhackl selftest` subcommand executes a small number of tests to verify
whether the system is configured correctly.


## Configuration

The configuration for the `baamhackl watch` subcommand is either specified via
the `-config` flag or the `BAAMHACKL_CONFIG_FILE` environment variable. The
following commands are equivalent:

* `baamhackl watch -config ./config.yaml`
* `BAAMHACKL_CONFIG_FILE=./config.yaml baamhackl watch`

Configuration files use the [YAML format](https://en.wikipedia.org/wiki/YAML).
At the root is a single option, `handlers`, which is a list of handler
configuration objects. Each handler supports the following options:

| Option | Default | Description |
| --- | --- | --- |
| `name` | *(none)* | Handler name. Used for logging and naming the trigger command in Watchman. |
| `path` | *(none)* | Absolute path to observed directory. |
| `command` | *(none)* | [Handler command](#handler-command) arguments as a list, e.g. `["/usr/local/bin/handle-change", "arg", "another"]`. Arguments are visible in log files and should not contain confidential information such as passwords or access tokens. Store them in separate files outside `path`. |
| `timeout` | `1h` | Timeout for executing the command. |
| `recursive` | `false` | Observe directory recursively (excluding the infrastructure directories). |
| `include_hidden` | false | Whether to invoke command for files starting with a dot (`.`). |
| `min_size_bytes`<br>`max_size_bytes` | 0 | Minimum and maximum file size for running command. Use zero to disable. Files smaller or larger than the configured values are ignored. |
| `settle_duration` | `1s` | Amount of time the filesystem should be idle before dispatching commands. |
| `retry_count` | 2 | Number of times a failing command should be retried. Set to 0 to make the first failure permanent. |
| `retry_delay_initial` | `15m` | Amount of time to wait between retry attempts. A small and random amount of variation is always applied. |
| `retry_delay_factor` | 1.5 | Back-off factor to apply between attempts after the first retry. Use 1 to always use the same delay. |
| `retry_delay_max` | `1h` | Maximum amount of time to wait between retry attempts. Use 0s for no limit. |
| `journal_dir` | `_/journal` | Path[^pathdirs] to directory for command logs. |
| `journal_retention` | 7 days | Amount of time before logs and processed files are deleted. |
| `success_dir` | `_/success` | Path[^pathdirs] to directory into which successfully handled files are moved. |
| `failure_dir` | `_/failure` | Path[^pathdirs] to directory for files for which the command failed persistently. |

[^pathdirs]: Relative paths are interpreted relative to the `path` option.
  Absolute paths are also supported. Directories beneath `path` are
  automatically created if necessary. All paths for a handler must reside on
  the same filesystem for atomic file moves.


## Handler command

Handler commands are started when a file change is detected. Commands are
considered to be successful when they exit with a zero status code. In all
other cases the command is re-run until it either succeeds or `retry_count`
attempts have passed.

Environment variables available to handler commands:

| Name | Description |
| --- | --- |
| `BAAMHACKL_PROGRAM` | Absolute path to the Baamhackl program. |
| `BAAMHACKL_ORIGINAL` | Path of changed file. Use only for informative purposes as the original may be modified concurrently. A copy of the file is made available via `BAAMHACKL_INPUT`. |
| `BAAMHACKL_INPUT` | Path to a copy of the changed file. |
| `BAAMHACKL_WORKDIR` | Path to a directory where the handler command can store temporary files. This is also the working directory when the command is started. |

If a command should produce an output in a particular directory it needs to do
so on its own. Baamhackl provides the `baamhackl move-into` subcommand to move
a file into a destination folder without overwriting any existing file. It does
so by finding a new and available name in case of a conflict. Example:

```shell
${BAAMHACKL_PROGRAM} move-into /srv/shared/finished ./output.pdf
```


## Installation

[Watchman][watchman] is a required dependency. By default the `watchman`
program is looked up via `$PATH`. Specify an absolute path using the
`-watchman_program` flag, e.g.
`baamhackl watch -watchman_program=/opt/watchman/bin/watchman`.

Pre-built binaries are provided for [all releases][releases]:

* Binary archives (`.tar.gz`)
* Debian/Ubuntu (`.deb`)
* RHEL/Fedora (`.rpm`)

With the source being available it's also possible to produce custom builds
directly using [Go](https://go.dev/) or [GoReleaser](https://goreleaser.com/).

The current implementation the Baamhackl program relies on a few of
Linux-specific system calls such as
[`renameat2()`](https://manpages.debian.org/stable/manpages-dev/renameat2.2.en.html).
Support for more operating systems would require the implementation of
alternatives.


## Security considerations

In multi-user environments it's strongly recommended to run Baamhackl in
a container with limited filesystem visibility. Only the directories used by
the configuration and handler commands should be made available.

Operations on filesystems shared by multiple users, either locally or via
network protocols such as
[Network File System (NFS)](https://en.wikipedia.org/wiki/Network_File_System) or
[Server Message Block (SMB)](https://en.wikipedia.org/wiki/Server_Message_Block),
are prone to race conditions. Locking isn't supported universally and can't be
relied upon.

A program like Baamhackl which observes file changes before acting upon them
needs to account for concurrent changes. Source files modified while the
handler command runs will cause a failure and a subsequent retry. Atomic file
operations are used where possible.

It's unrealistic to avoid race conditions under the given conditions. After the
handler command is done the originally changed file needs to be taken out of
the input directory to not re-process it later. Given that the file has been
processed it could be removed. However, between the command finishing, checking
for changes and removing the file a user could modify it again. The subsequent
removal would cause a data loss. For this reason files are first moved to an
archive directory where they remain for some time.

Path traversals are another issue. Modified files could be replaced with
a symlink between Watchman reporting a change and Baamhackl actually getting
around to processing the file.

Commands can also be given inputs causing them to read arbitrary files and
either logging their contents or copying them to a location accessible to an
attacker. The handler command `["bash", "-c", "source $BAAMHACKL_INPUT"]`
implements direct remote code execution.

[watchman]: https://facebook.github.io/watchman/
[releases]: https://github.com/hansmi/baamhackl/releases/latest

<!-- vim: set sw=2 sts=2 et : -->
