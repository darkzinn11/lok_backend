package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type BaseHandler struct {
	validator *validator.Validate
}

func NewBaseHandler() *BaseHandler {
	return &BaseHandler{
		validator: validator.New(),
	}
}

func (h *BaseHandler) Success(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (h *BaseHandler) Error(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (h *BaseHandler) Decode(r *http.Request, dst interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return err
	}
	return h.validator.Struct(dst)
}

func (h *BaseHandler) NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
