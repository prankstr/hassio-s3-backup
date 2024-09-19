package webui

import (
	"embed"
	"fmt"
	"hassio-proton-drive-backup/internal/config"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

//go:embed dist
var staticFiles embed.FS

func prettyPrintRequest(w http.ResponseWriter, r *http.Request) {
	// Print the HTTP method
	fmt.Printf("Method: %s\n", r.Method)

	// Print the scheme (http/https)
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	fmt.Printf("Scheme: %s\n", scheme)

	// Print the host (authority)
	fmt.Printf("Host: %s\n", r.Host)

	// Print the full URL
	fmt.Printf("URL: %s\n", r.URL.String())

	// Print the path
	fmt.Printf("Path: %s\n", r.URL.Path)

	// Print the query parameters
	if len(r.URL.RawQuery) > 0 {
		fmt.Printf("Query Params: %s\n", r.URL.RawQuery)
	}

	// Print the headers
	fmt.Println("Headers:")
	for name, values := range r.Header {
		// Loop over all headers
		for _, value := range values {
			fmt.Printf("  %s: %s\n", name, value)
		}
	}

	// Print the body (if available)
	if r.Body != nil {
		fmt.Println("Body:")
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil {
			bodyString := string(bodyBytes)
			// Print body content up to a reasonable limit to avoid overwhelming output
			if len(bodyString) > 1000 {
				bodyString = bodyString[:1000] + "...(truncated)"
			}
			fmt.Println(bodyString)
			// Reset the body for further use since it's already read
			r.Body = io.NopCloser(strings.NewReader(bodyString))
		} else {
			fmt.Println("Error reading body:", err)
		}
	}

	// Print information about the remote address and protocol
	fmt.Printf("Remote Address: %s\n", r.RemoteAddr)
	fmt.Printf("Protocol: %s\n", r.Proto)

	// Print the content length
	if r.ContentLength > 0 {
		fmt.Printf("Content Length: %d bytes\n", r.ContentLength)
	}

	// Print the cookies
	fmt.Println("Cookies:")
	for _, cookie := range r.Cookies() {
		fmt.Printf("  %s: %s\n", cookie.Name, cookie.Value)
	}
}

// NewHandler creates a new handler for serving static content from a given directory
func NewHandler(config *config.Options) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prettyPrintRequest(w, r)
		// Extract the file path after "/assets/"
		filePath := strings.TrimPrefix(r.URL.Path, "/assets/")

		// Determine the protocol
		proto := "http"
		if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
			proto = forwardedProto
		} else if r.TLS != nil {
			proto = "https"
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
