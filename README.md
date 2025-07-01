# Johnny Decimal Daemon

A background tool that watches a directory and automatically organizes files according to the [Johnny Decimal](https://johnnydecimal.com/) system.

## Features

- **Watches a directory tree** for new files.
- **Moves files** into the correct Johnny Decimal folders based on filename.
- **Configurable** via YAML file (`.jd.yaml` by default, or set with the `JDD_CONFIG` environment variable).
- **Exclusion patterns** and dry-run mode supported.
- **Runs as a daemon or in the foreground** (configurable).

## Quick Start

1. **Create a config file** (example: `.jd.yaml`):

```yaml
root: "." # Directory to watch (relative to config file location)
log_level: "info" # Log level: debug, info, warn, error
exclude:
  - ".git/**"
  - "tmp/**"
dry_run: false # If true, no files will be moved
daemonize: false # Run in foreground (set to true to daemonize)
```

2. **Run the daemon:**

```sh
go run ./cmd/daemon
# or specify a config file:
JDD_CONFIG=./tmp/.jd.yaml go run ./cmd/daemon
```

## Installation

If you use [Nix](https://nixos.org/), you can install JDD with:

```sh
nix-shell -p jdd
```

Or add it to your environment:

```sh
nix-env -iA nixpkgs.jdd
```

You can also download pre-built binaries from the [GitHub Releases page](https://github.com/mahyarmirrashed/jdd/releases).

## Notes

- By default, the daemon watches the directory specified in `root`, resolved relative to the config file’s location.
- Exclude patterns use glob syntax.
- Log output goes to `jdd.log` if daemonized, otherwise to stdout.
