package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

const StreamRoutePrefix = "/stream/"

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func StreamRoutePattern(sourceName string) string {
	return StreamRoutePrefix + url.PathEscape(sourceName) + "/"
}

func BuildStreamURL(sourceName, relSlash string) string {
	return StreamRoutePrefix + url.PathEscape(sourceName) + "/" + encodeURLPath(relSlash)
}

func encodeURLPath(relSlash string) string {
	parts := strings.Split(relSlash, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
