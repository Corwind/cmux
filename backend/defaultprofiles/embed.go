package defaultprofiles

import "embed"

//go:embed *.sbpl
var Embedded embed.FS
