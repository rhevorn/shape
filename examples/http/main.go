package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rhevorn/shape"
)

type Request struct {
	Name string `json:"name" shape:"trim,notempty"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	var request Request
	err := shape.BindJSONReaderContext(r.Context(), &request, r.Body, shape.JSONOptions{
		DisallowUnknownFields: true,
		MaxBytes:              1 << 20,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, shape.ErrJSONTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(request)
}

func main() {
	http.HandleFunc("/", handler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
