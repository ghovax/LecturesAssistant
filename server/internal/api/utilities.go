package api

import (
	"encoding/json"
	"net/http"
)

// writeJSONResponse writes a JSON response to the ResponseWriter
func writeJSONResponse(responseWriter http.ResponseWriter, value interface{}) error {
	encoder := json.NewEncoder(responseWriter)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
