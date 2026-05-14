package handlers

import (
	"net/http"

	"lockcenter-backend/internal/application"
)

type BranchHandler struct {
	*BaseHandler
	branchService *application.BranchService
}

func NewBranchHandler(branchService *application.BranchService) *BranchHandler {
	return &BranchHandler{
		BaseHandler:   NewBaseHandler(),
		branchService: branchService,
	}
}

func (h *BranchHandler) List(w http.ResponseWriter, r *http.Request) {
	branches, err := h.branchService.List(r.Context())
	if err != nil {
		h.Error(w, "failed to load branches", http.StatusInternalServerError)
		return
	}

	h.Success(w, branches, http.StatusOK)
}
