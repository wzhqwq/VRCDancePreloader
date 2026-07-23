package live

import (
	"encoding/json"
	"net/http"
)

type Result[T any] struct {
	Ok      bool   `json:"ok"`
	Data    T      `json:"data"`
	Message string `json:"message"`
}

func writeCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding")
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	writeCORSHeaders(w)
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeOk(w http.ResponseWriter, v interface{}) {
	writeJSON(w, http.StatusOK, &Result[any]{Ok: true, Data: v})
}
