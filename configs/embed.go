// Package configs embeds the default gearup config files so the binary
// ships with usable defaults. These are extracted to the user's config
// directory on first run or via `gearup init`.
package configs

import "embed"

//go:embed *.yaml
var Defaults embed.FS
