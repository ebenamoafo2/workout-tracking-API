package utils

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Envelope map[string]any

func WriteJSON(w http.ResponseWriter, status int, data Envelope) {
	js, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		// marshalling failed — send a plain 500 and log it
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	js = append(js, '\n')
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(js); err != nil {
		// response already started, can't change status — just log
		log.Printf("writeJSON: failed to write response body: %v", err)
	}
}

func ReadIDParam(r *http.Request) (int64, error) {
	idParam := chi.URLParam(r, "id")

	if idParam == "" {
		return 0, errors.New("invalid id is parameter")
	}
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}
