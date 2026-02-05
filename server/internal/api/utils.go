package api

import (
	"encoding/json"
	"net/http"
)

// writeJSONResponse writes a JSON response to the ResponseWriter
func writeJSONResponse(w http.ResponseWriter, v interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}
