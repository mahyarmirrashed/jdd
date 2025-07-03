# Johnny Decimal Daemon

A background tool that watches a directory and automatically organizes files according to the [Johnny Decimal](https://johnnydecimal.com/) system.

## Features

- **Watches a directory tree** for new files.
- **Moves files** into the correct Johnny Decimal folders based on filename.
- **Fully configurable** via command line arguments, environment variables, or an optional YAML config file (`.jd.yaml` by default).
- **Exclusion patterns** and dry-run mode supported.
- **Runs as a daemon or in the foreground** (configurable).

## Quick Start

You can run the daemon with **just command line flags or environment variables**. A config is not required.

### Example: Run with command line flags

```sh
jdd --root ~/Documents --log-level debug --dry-run
```

### Example: Run with environment variables

```sh
export JDD_ROOT=~/Documents
export JDD_LOG_LEVEL=info
jdd
```

### Example: Run with a config file (optional)

Create a config file (e.g., `.jd.yaml`):

```yaml
root: "." # Directory to watch (relative to config file location)
log_level: "info" # Log level: debug, info, warn, error
exclude:
  - ".git/**"
  - "tmp/**"
dry_run: false # If true, no files will be moved
daemonize: false # Run in foreground (set to true to daemonize)
delay: 1s # Duration to wait before processing new files
```

Then run:

```sh
jdd --config .jd.yaml
```

Or let it pick up the default `.jd.yaml` in the current directory.

## Installation

### Install with Nix

If you use [Nix](https://nixos.org/), you can install JDD with:

```sh
nix-shell -p jdd
```

Or add it to your environment:

```sh
nix-env -iA nixpkgs.jdd
```

### Install with Go

If you have Go 1.17+ installed, you can install JDD directly from the command line:

```sh
go install github.com/mahyarmirrashed/jdd@latest
```

### Pre-built Binaries

You can also download pre-built binaries from the [GitHub Releases page](https://github.com/mahyarmirrashed/jdd/releases).

## Notes

- The config file is **optional**&mdash;all settings can be provided via CLI flags or environment variables.
- By default, the daemon watches the directory specified in `root`, resolved relative to the config file’s location (if used), or as given by the flag/env.
- Exclude patterns use glob syntax.
- Log output goes to `jdd.log` if daemonized, otherwise to stdout.
