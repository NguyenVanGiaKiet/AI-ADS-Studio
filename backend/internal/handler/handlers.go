package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

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
	voices, err := h.TTSService.GetVoices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
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

func (h *Handler) DownloadTaskOutputs(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	temporaryFile, err := os.CreateTemp("", "remix-videos-*.zip")
	if err != nil {
		http.Error(w, "Không thể tạo tệp nén video: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(temporaryFile.Name())
	defer temporaryFile.Close()

	if err := h.RemixService.WriteTaskOutputsZip(taskID, temporaryFile); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrTaskOutputsNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, "Không thể tải các video của tiến trình: "+err.Error(), status)
		return
	}
	info, err := temporaryFile.Stat()
	if err != nil {
		http.Error(w, "Không thể đọc tệp nén video: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := temporaryFile.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "Không thể đọc lại tệp nén video: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="remix-videos.zip"`)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	if _, err := io.Copy(w, temporaryFile); err != nil {
		log.Printf("[Handler] Could not stream task output archive: %v", err)
	}
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

func (h *Handler) DownloadOutput(w http.ResponseWriter, r *http.Request) {
	outputID := r.PathValue("id")
	for _, output := range h.RemixService.GetOutputs() {
		if output.ID != outputID {
			continue
		}
		filename := filepath.Base(output.Filename)
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
			"filename": filename,
		}))
		http.ServeFile(w, r, filepath.Join(h.RemixService.OutputDir, filename))
		return
	}
	http.NotFound(w, r)
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
