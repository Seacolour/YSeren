package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// ErrorResponse 统一的错误响应格式
type ErrorResponse struct {
	Error string `json:"error"`
}

// WriteError 写入统一格式的 JSON 错误响应
func WriteError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: msg})
}

func streamRoutePattern(sourceName string) string {
	return "/stream/" + sourceName + "/"
}

func buildStreamURL(sourceName, relSlash string) string {
	return "/stream/" + url.PathEscape(sourceName) + "/" + encodeURLPath(relSlash)
}

// encodeURLPath：对每个 path segment 做 PathEscape，保留 “/” 作为层级分隔符。
func encodeURLPath(relSlash string) string {
	relSlash = strings.TrimPrefix(relSlash, "/")
	parts := strings.Split(relSlash, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}
