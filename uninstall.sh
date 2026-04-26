#!/usr/bin/env bash
# Uninstall gearup: remove the binary and (optionally) user configs/logs.
#
# Defaults:
#   - Remove the binary at $GEARUP_INSTALL_DIR/gearup (default ~/.local/bin/gearup).
#   - Leave user configs and logs alone.
#
# Env overrides:
#   GEARUP_INSTALL_DIR  Where the binary lives. Default: ~/.local/bin
#   GEARUP_PURGE        If set to 1, also remove user configs and logs without prompting.
#                       Honors XDG_CONFIG_HOME and XDG_STATE_HOME.
#
# Run interactively to be prompted before removing configs/logs:
#   curl -fsSL https://raw.githubusercontent.com/danlourenco/gearup/main/uninstall.sh | bash
#
# Or non-interactively (purge everything):
#   GEARUP_PURGE=1 curl -fsSL https://raw.githubusercontent.com/danlourenco/gearup/main/uninstall.sh | bash
set -euo pipefail

INSTALL_DIR="${GEARUP_INSTALL_DIR:-$HOME/.local/bin}"
TARGET="$INSTALL_DIR/gearup"

CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}"
CONFIG_DIR="$CONFIG_HOME/gearup"

STATE_HOME="${XDG_STATE_HOME:-$HOME/.local/state}"
LOG_DIR="$STATE_HOME/gearup"

removed_anything=0

# 1. Remove the binary.
if [ -f "$TARGET" ] || [ -L "$TARGET" ]; then
  rm -f "$TARGET"
  echo "Removed $TARGET"
  removed_anything=1
else
  echo "No binary at $TARGET — skipping."
fi

# 2. Remove user configs (with confirmation unless GEARUP_PURGE=1).
purge_dir() {
  local dir="$1"
  local label="$2"
  if [ ! -d "$dir" ]; then
    return
  fi

  if [ "${GEARUP_PURGE:-0}" = "1" ]; then
    rm -rf "$dir"
    echo "Removed $label at $dir"
    removed_anything=1
    return
  fi

  # Only prompt if we have a TTY (interactive shell, not piped via curl | bash).
  if [ -t 0 ]; then
    read -r -p "Remove $label at $dir? [y/N] " ans
    case "$ans" in
      y|Y|yes|YES)
        rm -rf "$dir"
        echo "Removed $label at $dir"
        removed_anything=1
        ;;
      *)
        echo "Kept $label at $dir"
        ;;
    esac
  else
    echo "Kept $label at $dir (re-run with GEARUP_PURGE=1 to remove)"
  fi
}

purge_dir "$CONFIG_DIR" "user configs"
purge_dir "$LOG_DIR"    "logs"

if [ "$removed_anything" = "0" ]; then
  echo
  echo "Nothing to remove. gearup does not appear to be installed."
  exit 0
fi

echo
echo "gearup uninstalled."
echo "If you added $INSTALL_DIR to your PATH explicitly, you may want to remove that line from your shell rc."
