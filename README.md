# gearup

An open-source, config-driven macOS developer-machine bootstrap CLI built with [Bun](https://bun.sh) and the [unjs](https://unjs.io) ecosystem ([c12](https://github.com/unjs/c12), [confbox](https://github.com/unjs/confbox), [citty](https://github.com/unjs/citty)) plus [Valibot](https://valibot.dev), [execa](https://github.com/sindresorhus/execa), and [Clack](https://github.com/natemoo-re/clack).

Define your team's toolchain in a JSONC, YAML, or TOML config, run `gearup run`, and get a fully provisioned dev machine in minutes. Every step is idempotent — re-running skips what's already installed.

## Quick start

### Install (no Bun required)

```bash
curl -fsSL https://raw.githubusercontent.com/danlourenco/gearup/main/install.sh | bash
```

Detects your architecture (`arm64` / `x64`), downloads the latest Bun-compiled binary to `~/.local/bin/`, verifies the SHA256 checksum, and installs. No `sudo` required.

To install to a different location:

```bash
GEARUP_INSTALL_DIR=~/bin curl -fsSL https://raw.githubusercontent.com/danlourenco/gearup/main/install.sh | bash
```

Default configs (`base`, `backend`, `frontend`, etc.) are embedded in the binary. On first run, they're automatically extracted to `~/.config/gearup/configs/`. To reset defaults or refresh them after a release:

```bash
gearup init          # write any missing defaults; preserve customizations
gearup init --force  # overwrite all customizations with the embedded defaults
```

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/danlourenco/gearup/main/uninstall.sh | bash
```

Removes the binary at `~/.local/bin/gearup` (or `$GEARUP_INSTALL_DIR`). User configs in `~/.config/gearup/` and logs in `~/.local/state/gearup/` are kept by default. Pipe-to-bash isn't interactive, so to remove everything pass `GEARUP_PURGE=1`:

```bash
GEARUP_PURGE=1 curl -fsSL https://raw.githubusercontent.com/danlourenco/gearup/main/uninstall.sh | bash
```

Or run interactively (clone the repo first or save the script locally) to be prompted per directory.

### Pick and run interactively

```bash
gearup run    # presents a picker of available configs, then runs the chosen one
```

Or specify directly:

```bash
gearup run --config ~/.config/gearup/configs/backend.jsonc
gearup plan --config ~/.config/gearup/configs/backend.jsonc   # check-only
```

### Build from source (requires Bun)

```bash
git clone https://github.com/danlourenco/gearup.git
cd gearup
bun install
bun run src/cli.ts version
bun build src/cli.ts --compile --outfile=bin/gearup   # build local binary
```

## How it works

```
argv → CLI router (citty) → Config loader (c12 + confbox) → Validator (Valibot)
       → Runner → Handler registry → exec (execa, wrapped for logging)
```

Configs are JSONC/YAML/TOML files with an `extends:` array for composition. Each config has a `name`, optional `description`, optional `platform` constraints, optional `elevation:` block, and a `steps` map keyed by step name.

When you run `gearup plan` or `gearup run` and the chosen config has `extends:`, [c12](https://github.com/unjs/c12) recursively loads each referenced config and deep-merges them with [defu](https://github.com/unjs/defu) defaults. **On step name collisions, the current (override) config wins.**

### Picker resolution

When `--config` is omitted, gearup discovers configs in two locations:

1. `~/.config/gearup/configs/` (or `$XDG_CONFIG_HOME/gearup/configs/`) — user-global
2. `./configs/` (relative to your current working directory) — project-local

Configs are presented as a flat union. **On name collision, the project-local config wins** (closer to your `pwd`, presumably more relevant). Each entry shows its `name`, `description`, and source label (`[user]` or `[project]`).

If both locations are empty on first run, gearup auto-extracts the embedded defaults to the user dir before prompting.

### Example config (entry-point)

```jsonc
// ~/.config/gearup/configs/backend.jsonc
{
  "version": 1,
  "name": "Backend",
  "description": "Full macOS developer toolchain for backend/infra work",
  "platform": { "os": ["darwin"] },
  "elevation": {
    "message": "Some steps need admin permissions. Elevate now, then continue.",
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

Note: extends array entries **must include the file extension** for non-JS configs. `./base.jsonc` works; `./base` does not.

### Example reusable config

```jsonc
// ~/.config/gearup/configs/base.jsonc
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

`extends:` and `steps:` can coexist — extended configs' steps are merged in first, then the current config's steps are layered on top with override semantics.

## Step types

Each step in `steps` declares a `type` that determines how it's installed and checked.

| Type | Installs via | Auto-check | Explicit `check:` |
|---|---|---|---|
| `brew` | `brew install <formula>` | `brew list --formula <formula>` | Optional override |
| `brew-cask` | `brew install --cask <cask>` | `brew list --cask <cask>` | Optional override |
| `curl-pipe-sh` | `curl -fsSL <url> \| <shell>` | None | **Required** |
| `git-clone` | `git clone [--branch <ref>] <repo> <dest>` | Directory exists at `dest` | Not needed |
| `shell` | User-provided `install:` command | None | **Required** |

Every step is idempotent: the `check` command runs first, and if it exits 0 the install is skipped.

Any step type can include `post_install:` — a list of shell commands that run after a successful install (skipped if the step was already installed):

```jsonc
{
  "Colima": {
    "type": "brew",
    "formula": "colima",
    "post_install": ["colima start --cpu 4 --memory 8"]
  }
}
```

## Commands

```
gearup plan     [--config <path>]
gearup run      [--config <path>]
gearup init     [--force]
gearup version
```

### `gearup plan`

Runs every step's `check` without installing anything. Prints a styled preview showing what would happen.

Exit codes:
- `0` — machine is fully provisioned (nothing would run)
- `10` — one or more steps would install (CI-friendly: "machine not up to date")

### `gearup run`

Runs `check` per step; if a step is not installed, runs its install. Then runs `post_install` commands if any. Stops on first error (fail-fast). Steps with `requires_elevation: true` run first if the config has an `elevation:` block — gearup shows a confirmation banner before they run.

### `gearup init`

Writes the embedded default configs to `~/.config/gearup/configs/`. Existing files are preserved unless `--force` is passed. Useful for resetting to defaults or seeing what configs ship with the binary.

### Flags

| Flag | Description |
|---|---|
| `--config <path>` | Path to config (JSONC/YAML/TOML; extension-less paths are also accepted). Omit to pick interactively. |
| `--force` | (`init` only) Overwrite existing files. |

## Elevation

Steps that need admin permissions declare `requires_elevation: true`. When a config includes an `elevation:` block, gearup shows a styled banner and waits for confirmation before running those steps:

```jsonc
{
  "elevation": {
    "message": "Some steps need admin permissions. Elevate now, then continue.",
    "duration": "180s"
  }
}
```

gearup never invokes elevation itself — it pauses and lets you acquire permissions through whatever mechanism your organization uses (MDM scripts, Touch ID, native sudo, etc.). If no `elevation:` block is set, steps that need sudo prompt natively.

**Smart suppression:** if all elevation-required steps are already installed, the banner is skipped entirely.

## Log files

Every `gearup run` writes a log file at:

```
$XDG_STATE_HOME/gearup/logs/<YYYYMMDD>-<HHMMSS>-<config>.log
```

(Falls back to `~/.local/state/gearup/logs/` if `XDG_STATE_HOME` is unset.)

Each subprocess invocation is captured: argv, exit code, duration, full stdout, full stderr. The terminal stays clean — only step status (via Clack spinner) and the log path are shown. On failure, the relevant captured output is shown alongside the log path.

## Creating your own configs

Run `gearup init` to extract the default configs at `~/.config/gearup/configs/`. Edit them directly, or use them as templates:

1. Pick a directory for your configs (the user dir, or `./configs/` in your project for team-shared).
2. Write one config file per concern (base tools, language runtimes, cloud tooling, etc.) in JSONC, YAML, or TOML.
3. Create an entry-point config that lists `extends:` with explicit paths and extensions:
   ```jsonc
   {
     "version": 1,
     "name": "my-stack",
     "extends": [
       "./base.jsonc",
       "./node.jsonc"
     ]
   }
   ```
4. Run `gearup run --config <your-config.jsonc>` (or just `gearup run` and pick from the list).

`extends:` accepts:
- **Relative paths**: `./base.jsonc` (relative to the current config file)
- **Absolute paths**: `/path/to/base.jsonc`
- **`github:` references** (dev mode only): `github:owner/repo/path/to/file.jsonc`
- **`https:` URLs** (dev mode only): `https://example.com/configs/base.jsonc`

For non-JS configs (jsonc/yaml/toml), the **file extension is required** in extends entries.

## Running tests

```bash
bun test
bun run typecheck
```

## License

[MIT](LICENSE)
