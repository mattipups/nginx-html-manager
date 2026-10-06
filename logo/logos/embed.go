package branding

import "embed"

// Assets embeds the original Light/Dark PNGs, identical to the supplied files.
//go:embed builder-light-1400x700.png builder-dark-1400x700.png
var Assets embed.FS
