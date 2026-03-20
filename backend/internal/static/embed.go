package static

import "embed"

//go:embed dist/*
var Assets embed.FS
