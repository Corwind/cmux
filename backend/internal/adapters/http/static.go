package http

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// SPAHandler serves static files from a filesystem with SPA fallback.
// Any request that doesn't match a real file is served index.html.
func SPAHandler(assets fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(assets))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean the path and try to open the file
		p := path.Clean(r.URL.Path)
		if p == "/" {
			p = "index.html"
		} else {
			p = strings.TrimPrefix(p, "/")
		}

		// Check if the file exists
		if _, err := fs.Stat(assets, p); err == nil {
			// Hashed assets (js, css with content hashes) get long cache
			if strings.HasPrefix(p, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for any non-file path
		r.URL.Path = "/"
		w.Header().Set("Cache-Control", "no-cache")
		fileServer.ServeHTTP(w, r)
	})
}
