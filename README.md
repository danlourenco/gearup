# gearup

An open-source, config-driven macOS developer-machine bootstrap CLI built with [Go](https://go.dev) and the [Charm](https://charm.sh) ecosystem.

Define your team's toolchain in a YAML config, run `gearup run`, and get a fully provisioned dev machine in minutes. Every step is idempotent — re-running skips what's already installed.

## Quick start

### Install from GitHub releases (no Go required)

```bash
# macOS (Apple Silicon)
curl -fsSL https://github.com/danlourenco/gearup/releases/latest/download/gearup_darwin_arm64.tar.gz | tar xz
sudo mv gearup /usr/local/bin/

# macOS (Intel)
curl -fsSL https://github.com/danlourenco/gearup/releases/latest/download/gearup_darwin_amd64.tar.gz | tar xz
sudo mv gearup /usr/local/bin/
```

### Build from source (requires Go 1.22+)

```bash
go build -o gearup ./cmd/gearup

# Pick a config interactively
./gearup run

# Or specify one directly
./gearup run --config ./examples/configs/backend.yaml
```

On first run you'll see an interactive picker listing discovered configs. Select one, and gearup walks through each step with an animated progress indicator:

```
CONFIG: Backend  (12 steps)

  ✓ [1/12] Homebrew  already installed
  ✓ [2/12] Git  already installed
  ⠋ [3/12] jq  installing... 2.1s
  ✓ [3/12] jq  installed (3.6s)
  ✓ [4/12] nvm  already installed
  ...

Done.
Log: ~/.local/state/gearup/logs/20260415-211527-Backend.log
```

## How it works

Every gearup YAML file has the same shape — there is no distinction between an "entry point" and a "reusable component." A config can declare steps directly, extend other configs by name, or both.

Composition is via `extends: [name, ...]`. The config file's own directory is always in the search path, so configs in the same directory can extend each other without any extra declaration.

```
examples/configs/
├── backend.yaml       ← extends: [base, jvm, containers, aws-k8s, node]
├── frontend.yaml      ← extends: [base, node]
├── base.yaml          (Homebrew, Git, jq)
├── jvm.yaml           (OpenJDK 21 + system symlink)
├── containers.yaml    (Docker CLI, Compose plugin, Colima)
├── aws-k8s.yaml       (AWS CLI, aws-iam-authenticator, kubectl)
└── node.yaml          (nvm)
```

### Example config

```yaml
# configs/backend.yaml
version: 1
name: "Backend"
description: "Full macOS developer toolchain for backend/infra work"

platform:
  os: [darwin]

elevation:
  message: "Some steps need admin permissions. Elevate now, then press Continue."
  duration: 180s

extends:
  - base
  - jvm
  - containers
  - aws-k8s
  - node
```

No `sources:` declaration needed — configs in the same directory find each other automatically.

### Example reusable config

```yaml
# configs/base.yaml
version: 1
name: base
description: "Homebrew + universal core CLI tools (git, jq)"

steps:
  - name: Homebrew
    type: curl-pipe-sh
    url: https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh
    check: command -v brew

  - name: Git
    type: brew
    formula: git
    check: command -v git    # skip if git exists from any source

  - name: jq
    type: brew
    formula: jq
```

## Step types

Each step in a config declares a `type` that determines how it's installed and checked.

| Type | Installs via | Auto-check | Explicit `check:` |
|---|---|---|---|
| `brew` | `brew install <formula>` | `brew list --formula <formula>` | Optional override (e.g., `command -v git`) |
| `curl-pipe-sh` | `curl -fsSL <url> \| bash` | None | **Required** |
| `shell` | User-provided `install:` command | None | **Required** |

Every step is idempotent: the `check` command runs first, and if it exits 0 the install is skipped.

## Commands

```
gearup run      [--config <path>] [--dry-run] [--yes]
gearup plan     [--config <path>]
gearup version
```

### `gearup run`

Executes a config. Without `--config`, discovers configs in `$XDG_CONFIG_HOME/gearup/configs/` and `./examples/configs/` and shows an interactive picker.

### `gearup plan`

Alias for `gearup run --dry-run`. Runs every step's check without installing anything. Prints a styled preview showing what would happen.

Exit codes:
- `0` — machine is fully provisioned (nothing would run)
- `10` — one or more steps would install (CI-friendly: "machine not up to date")

### Flags

| Flag | Description |
|---|---|
| `--config <path>` | Path to config YAML. Omit to pick interactively. |
| `--dry-run` | Check only, don't install. Exit 10 if anything would run. |
| `--yes` | Auto-approve the elevation confirmation prompt (for scripted/CI use). |

## Elevation

Steps that need admin permissions declare `requires_elevation: true`. When a config includes an `elevation:` block, gearup shows a styled banner and waits for you to confirm before running those steps:

```yaml
elevation:
  message: "Some steps need admin permissions. Elevate now, then press Continue."
  duration: 180s    # advisory countdown
```

gearup never invokes elevation itself — it pauses and lets you acquire permissions through whatever mechanism your organization uses (MDM scripts, Touch ID, native sudo, etc.). If no `elevation:` block is set, steps that need sudo prompt natively.

Smart suppression: if all elevation-required steps are already installed, the banner is skipped entirely.

## Log files

Every `gearup run` writes a log file at:

```
$XDG_STATE_HOME/gearup/logs/<timestamp>-<config>.log
```

(Falls back to `~/.local/state/gearup/logs/` if `XDG_STATE_HOME` is unset.)

Check and install command output is captured here. The terminal stays clean — only step status lines and the log path are shown. On failure, the relevant captured output is printed inline alongside the log path.

## Creating your own configs

1. Create a directory for your configs.
2. Write one YAML file per concern (base tools, language runtimes, cloud tooling, etc.).
3. Create an entry-point config that lists `extends: [name, ...]` (names of files to include, without `.yaml`).
4. Run `gearup run --config <your-config.yaml>`.

Configs can live anywhere — a local directory, a shared team repo, or all in the same folder. If the extended configs are in a different location, declare it with `sources`:

```yaml
# my-team-config.yaml
version: 1
name: "My Team"
sources:
  - path: ./my-configs
  - path: ~/src/shared-configs
extends:
  - base
  - my-custom-stack
```

When configs are in the same directory, no `sources` declaration is needed.

## Building from source

Requires Go 1.22 or later.

```bash
git clone https://github.com/danlourenco/gearup.git
cd gearup
go build -o gearup ./cmd/gearup
./gearup version
```

## Running tests

```bash
go test ./...
```

## License

[MIT](LICENSE)
