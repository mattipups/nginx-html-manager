package favicons

import "embed"

// Assets embeds the existing repository favicons without changing their bytes.
//go:embed favicon.ico favicon-16.png favicon-32.png favicon-48.png
var Assets embed.FS
