package web

import (
	"encoding/json"
	"net/http"
)

func writeJson(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeJsonError(w http.ResponseWriter, status int, code, message string) {
	writeJson(w, status, map[string]string{"error": code, "message": message})
}
