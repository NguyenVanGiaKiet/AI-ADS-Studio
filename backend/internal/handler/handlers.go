package handler

import (
	"encoding/json"
	"net/http"

	"ai-ads-studio/backend/internal/model"
	"ai-ads-studio/backend/internal/service"
)

type Handler struct {
	RemixService *service.RemixService
	TTSService   *service.TTSService
}

func NewHandler(remixSvc *service.RemixService, ttsSvc *service.TTSService) *Handler {
	return &Handler{
		RemixService: remixSvc,
		TTSService:   ttsSvc,
	}
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "AI ADS Studio Backend (Go)",
	})
}

func (h *Handler) UploadVideo(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(500 << 20); err != nil { // 500 MB max
		http.Error(w, "Cannot parse form or file too large: "+err.Error(), http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		// fallback to single "file"
		files = r.MultipartForm.File["file"]
	}

	if len(files) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	var uploaded []model.UploadedVideo
	for _, fileHeader := range files {
		item, err := h.RemixService.SaveUpload(fileHeader)
		if err != nil {
			http.Error(w, "Error saving file "+fileHeader.Filename+": "+err.Error(), http.StatusInternalServerError)
			return
		}
		uploaded = append(uploaded, *item)
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"message": "Upload successful",
		"data":    uploaded,
	})
}

func (h *Handler) GetUploads(w http.ResponseWriter, r *http.Request) {
	uploads := h.RemixService.GetUploads()
	respondJSON(w, http.StatusOK, map[string]any{
		"data": uploads,
	})
}

func (h *Handler) GetVoices(w http.ResponseWriter, r *http.Request) {
	voices := h.TTSService.GetVoices()
	respondJSON(w, http.StatusOK, map[string]any{"data": voices})
}

func (h *Handler) PreviewTTS(w http.ResponseWriter, r *http.Request) {
	var req model.TTSPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	resp, err := h.TTSService.GeneratePreview(req)
	if err != nil {
		http.Error(w, "TTS Preview error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data": resp,
	})
}

func (h *Handler) CreateRemixTask(w http.ResponseWriter, r *http.Request) {
	var req model.RemixRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	task, err := h.RemixService.CreateTask(req)
	if err != nil {
		http.Error(w, "Error creating remix task: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]any{
		"message": "Remix task created",
		"data":    task,
	})
}

func (h *Handler) GetTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		taskID = r.URL.Query().Get("id")
	}

	task, ok := h.RemixService.GetTask(taskID)
	if !ok {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data": task,
	})
}

func (h *Handler) GetTasks(w http.ResponseWriter, r *http.Request) {
	tasks := h.RemixService.GetTasks()
	respondJSON(w, http.StatusOK, map[string]any{
		"data": tasks,
	})
}

func (h *Handler) GetOutputs(w http.ResponseWriter, r *http.Request) {
	outputs := h.RemixService.GetOutputs()
	respondJSON(w, http.StatusOK, map[string]any{
		"data": outputs,
	})
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
