package components

import "embed"

//go:embed */*.js */*.css
var Assets embed.FS
