package static

import "embed"

//go:embed all:dist
var Assets embed.FS
