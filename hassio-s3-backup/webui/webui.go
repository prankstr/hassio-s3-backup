package webui

import (
	"embed"
	"hassio-proton-drive-backup/internal/config"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

//go:embed dist
var staticFiles embed.FS

// NewHandler creates a new handler for serving static content from a given directory
func NewHandler(config *config.Options) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the file path after "/assets/"
		filePath := strings.TrimPrefix(r.URL.Path, "/assets/")

		// Determine the protocol
		proto := "http"
		if r.TLS != nil {
			proto = "https"
		} else if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
			proto = forwardedProto
		}

		// Extract the host from the incoming request
		host, port, err := net.SplitHostPort(r.Host)
		if err != nil {
			// If the host is not in the form "host:port", use the Host value directly (without port).
			host = r.Host
		} else {
			// If a port is present, append it to the host
			host = net.JoinHostPort(host, port)
		}

		selfUrl := proto + "://" + host + config.IngressPath

		// Check if the request is for assets
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			// Read the file from the embedded filesystem
			fileContent, err := staticFiles.ReadFile("dist/assets/" + filePath)
			if err != nil {
				http.NotFound(w, r)
				return
			}

			// Set headers
			w.Header().Set("Content-Type", "image/svg+xml")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

			// Serve the file content
			w.Write(fileContent)
			return
		}

		// Handle other routes (e.g., "/")
		modifiedData := modifyContent(staticFiles, selfUrl)
		http.ServeContent(w, r, "index.html", time.Now(), strings.NewReader(string(modifiedData)))
	})
}

func modifyContent(htmlContent fs.FS, url string) []byte {
	// Read the original content
	originalData, err := fs.ReadFile(htmlContent, "dist/index.html")
	if err != nil {
		log.Fatal(err)
	}

	// Modify the content
	modifiedData := strings.ReplaceAll(string(originalData), "http://replaceme.homeassistant", url)
	modifiedData = strings.ReplaceAll(modifiedData, "/assets/", url+"/assets/")

	return []byte(modifiedData)
}
