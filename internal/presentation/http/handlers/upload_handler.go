package handlers

import (
	"net/http"
	"lockcenter-backend/internal/application"
)

type UploadHandler struct {
	*BaseHandler
	imageService *application.ImageService
}

func NewUploadHandler(imageService *application.ImageService) *UploadHandler {
	return &UploadHandler{
		BaseHandler:  NewBaseHandler(),
		imageService: imageService,
	}
}

func (h *UploadHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	// 1. Limit upload size (e.g., 10MB)
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)
	
	err := r.ParseMultipartForm(10 * 1024 * 1024)
	if err != nil {
		h.Error(w, "file too large or invalid form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	folder := r.FormValue("folder")
	if folder == "" {
		folder = "uploads"
	}

	// 2. Process and Upload via ImageService (this will compress and resize)
	result, err := h.imageService.ProcessAndUpload(r.Context(), file, header.Filename, folder)
	if err != nil {
		h.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Success(w, result, http.StatusOK)
}
