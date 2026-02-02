package main

import (
	"encoding/json"
	"net/http"
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
