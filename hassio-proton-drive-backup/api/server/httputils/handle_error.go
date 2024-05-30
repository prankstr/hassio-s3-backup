package httputils

import (
	"fmt"
	"net/http"
)

// handleError handles errors by logging them and writing an error response to the client
func HandleError(w http.ResponseWriter, err error, statusCode int) {
	fmt.Println("Error:", err)
	http.Error(w, err.Error(), statusCode)
}
