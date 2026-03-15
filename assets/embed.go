package assets

import "embed"

//go:embed all:css all:js all:static
var Files embed.FS
