package httputil

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type ErrorDetail struct {
	Error     string `json:"error"`
	Code      string `json:"code,omitempty"`
	Action    string `json:"action,omitempty"`
	ActionURL string `json:"action_url,omitempty"`
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

func WriteErrorDetail(w http.ResponseWriter, status int, detail ErrorDetail) {
	WriteJSON(w, status, detail)
}

func DecodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
