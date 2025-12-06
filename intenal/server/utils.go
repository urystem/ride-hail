package server

import (
	"encoding/json"
	"net/http"
)

type myErr struct {
	ErrStr string `json:"error"`
}

func errorWrite(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	msg := &myErr{
		ErrStr: err.Error(),
	}
	json.NewEncoder(w).Encode(msg)
}
