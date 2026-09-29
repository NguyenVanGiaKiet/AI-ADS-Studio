package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"ai-ads-studio/backend/internal/handler"
	"ai-ads-studio/backend/internal/service"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	baseStorageDir := filepath.Join(".", "storage")
	uploadDir := filepath.Join(baseStorageDir, "uploads")
	outputDir := filepath.Join(baseStorageDir, "outputs")
	ttsDir := filepath.Join(baseStorageDir, "tts")

	// Initialize services
	ffmpegSvc := service.NewFFmpegService(outputDir)
	ttsSvc := service.NewTTSService(ttsDir)
	remixSvc := service.NewRemixService(uploadDir, outputDir, ffmpegSvc, ttsSvc)

	// Initialize HTTP handlers
	h := handler.NewHandler(remixSvc, ttsSvc)

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("GET /api/health", h.HealthCheck)
	mux.HandleFunc("POST /api/upload", h.UploadVideo)
	mux.HandleFunc("GET /api/uploads", h.GetUploads)
	mux.HandleFunc("GET /api/tts/voices", h.GetVoices)
	mux.HandleFunc("POST /api/tts/preview", h.PreviewTTS)
	mux.HandleFunc("POST /api/remix/tasks", h.CreateRemixTask)
	mux.HandleFunc("GET /api/remix/tasks/{id}", h.GetTaskStatus)
	mux.HandleFunc("GET /api/remix/tasks", h.GetTasks)
	mux.HandleFunc("GET /api/videos", h.GetOutputs)

	// Static file serving for uploads, outputs, and TTS audio
	fs := http.FileServer(http.Dir(baseStorageDir))
	mux.Handle("GET /storage/", http.StripPrefix("/storage/", fs))

	handlerWithCORS := corsMiddleware(mux)

	log.Printf("🚀 AI ADS Studio Go Backend server listening on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, handlerWithCORS); err != nil {
		log.Fatalf("Server stopped with error: %v", err)
	}
}
