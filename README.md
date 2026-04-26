# gearup

An open-source, config-driven macOS developer-machine bootstrap CLI built with [Go](https://go.dev) and the [Charm](https://charm.sh) ecosystem.

Define your team's toolchain in a YAML config, run `gearup run`, and get a fully provisioned dev machine in minutes. Every step is idempotent — re-running skips what's already installed.

## Quick start

### Install (no Go required)

```bash
curl -fsSL https://raw.githubusercontent.com/danlourenco/gearup/master/install.sh | bash
```

Detects your architecture, downloads the latest release to `~/.local/bin/`, and verifies the install. No `sudo` required.

Default configs (backend, frontend, etc.) are embedded in the binary. On first `gearup run`, they're automatically extracted to `~/.config/gearup/configs/`. To reset defaults or see what's available:

```bash
gearup init          # write/refresh default configs
gearup init --force  # overwrite any customizations with defaults
```

To install to a different location:

```bash
GEARUP_INSTALL_DIR=~/bin curl -fsSL https://raw.githubusercontent.com/danlourenco/gearup/master/install.sh | bash
```

### Build from source (requires Go 1.22+)

```bash
go build -o gearup ./cmd/gearup

# Pick a config interactively
./gearup run

# Or specify one directly
./gearup run --config ./configs/backend.jsonc
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

Every gearup config file has the same shape — there is no distinction between an "entry point" and a "reusable component." A config can declare steps directly, extend other configs by path, or both.

Composition is via `extends: [path, ...]`. Each path is resolved relative to the config file. Extensions are required for non-JS configs (`./base.jsonc`, `./jvm.yaml`, etc.). c12 also supports npm package names and `github:owner/repo` references for shared team configs.

```
configs/
├── backend.jsonc       ← extends: [./base.jsonc, ./jvm.jsonc, ./containers.jsonc, ./aws-k8s.jsonc, ./node.jsonc]
├── frontend.jsonc      ← extends: [./base.jsonc, ./node.jsonc]
├── base.jsonc          (Homebrew, Git, jq)
├── jvm.jsonc           (OpenJDK 21 + system symlink)
├── containers.jsonc    (Docker CLI, Compose plugin, Colima)
├── aws-k8s.jsonc       (AWS CLI, aws-iam-authenticator, kubectl)
└── node.jsonc          (nvm)
```

### Example config

```jsonc
// configs/backend.jsonc
{
  "version": 1,
  "name": "Backend",
  "description": "Full macOS developer toolchain for backend/infra work",

  "platform": {
    "os": ["darwin"]
  },

  "elevation": {
    "message": "Some steps need admin permissions. Elevate now, then press Continue.",
    "duration": "180s"
  },

  "extends": [
    "./base.jsonc",
    "./jvm.jsonc",
    "./containers.jsonc",
    "./aws-k8s.jsonc",
    "./node.jsonc"
  ]
}
```

### Example reusable config

```jsonc
// configs/base.jsonc
{
  "version": 1,
  "name": "base",
  "description": "Homebrew + universal core CLI tools (git, jq)",

  "steps": {
    "Homebrew": {
      "type": "curl-pipe-sh",
      "url": "https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh",
      "check": "command -v brew"
    },
    "Git": {
      "type": "brew",
      "formula": "git",
      "check": "command -v git"
    },
    "jq": {
      "type": "brew",
      "formula": "jq"
    }
  }
}
```

## Step types

Each step in a config declares a `type` that determines how it's installed and checked.

| Type | Installs via | Auto-check | Explicit `check:` |
|---|---|---|---|
| `brew` | `brew install <formula>` | `brew list --formula <formula>` | Optional override |
| `brew-cask` | `brew install --cask <cask>` | `brew list --cask <cask>` | Optional override |
| `curl-pipe-sh` | `curl -fsSL <url> \| bash` | None | **Required** |
| `git-clone` | `git clone <repo> <dest>` | Directory exists at `dest` | Not needed |
| `shell` | User-provided `install:` command | None | **Required** |

Every step is idempotent: the `check` command runs first, and if it exits 0 the install is skipped.

Any step type can include `post_install:` — a list of shell commands that run after a successful install (skipped if the step was already installed):

```yaml
steps:
  - name: Colima
    type: brew
    formula: colima
    post_install:
      - colima start --cpu 4 --memory 8
```

## Commands

```
gearup run      [--config <path>] [--dry-run] [--yes]
gearup plan     [--config <path>]
gearup init     [--force]
gearup version
```

### `gearup run`

Executes a config. Without `--config`, discovers configs in `$XDG_CONFIG_HOME/gearup/configs/` and `./configs/` and shows an interactive picker.

### `gearup plan`

Alias for `gearup run --dry-run`. Runs every step's check without installing anything. Prints a styled preview showing what would happen.

Exit codes:
- `0` — machine is fully provisioned (nothing would run)
- `10` — one or more steps would install (CI-friendly: "machine not up to date")

### `gearup init`

Writes the embedded default configs to `~/.config/gearup/configs/`. Existing files are preserved unless `--force` is passed. Useful for resetting to defaults or seeing what configs ship with the binary.

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

Start by running `gearup init` to see the default configs at `~/.config/gearup/configs/`. Edit them directly, or use them as templates:

1. Create a directory for your configs.
2. Write one config file per concern (base tools, language runtimes, cloud tooling, etc.) using JSONC, YAML, or TOML.
3. Create an entry-point config that lists `extends:` with explicit paths and extensions:
   ```jsonc
   {
     "version": 1,
     "name": "Backend",
     "extends": [
       "./base.jsonc",
       "./jvm.jsonc",
       "github:my-team/configs/aws.yaml"
     ]
   }
   ```
4. Run `gearup run --config <your-config.jsonc>`.

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
