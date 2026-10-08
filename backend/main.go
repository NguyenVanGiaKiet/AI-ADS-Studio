package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"ai-ads-studio/backend/internal/handler"
	"ai-ads-studio/backend/internal/service"
	"github.com/joho/godotenv"
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
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("Could not load backend/.env: %v", err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	outputDir := filepath.Join("storage", "outputs")
	temporaryDir, err := os.MkdirTemp("", "ai-ads-studio-")
	if err != nil {
		log.Fatalf("Could not create temporary storage: %v", err)
	}
	defer os.RemoveAll(temporaryDir)
	uploadDir := filepath.Join(temporaryDir, "uploads")
	ttsDir := filepath.Join(temporaryDir, "tts")

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
	mux.HandleFunc("GET /api/remix/tasks/{id}/download", h.DownloadTaskOutputs)
	mux.HandleFunc("GET /api/remix/tasks/{id}", h.GetTaskStatus)
	mux.HandleFunc("GET /api/remix/tasks", h.GetTasks)
	mux.HandleFunc("GET /api/videos/{id}/download", h.DownloadOutput)
	mux.HandleFunc("GET /api/videos/download", h.DownloadAllOutputs)
	mux.HandleFunc("POST /api/videos/download", h.DownloadSelectedOutputs)
	mux.HandleFunc("GET /api/videos", h.GetOutputs)

	// Keep transient media outside persistent storage while preserving the API URLs.
	mux.Handle("GET /storage/uploads/", http.StripPrefix("/storage/uploads/", http.FileServer(http.Dir(uploadDir))))
	mux.Handle("GET /storage/outputs/", http.StripPrefix("/storage/outputs/", http.FileServer(http.Dir(outputDir))))
	mux.Handle("GET /storage/tts/", http.StripPrefix("/storage/tts/", http.FileServer(http.Dir(ttsDir))))

	handlerWithCORS := corsMiddleware(mux)

	log.Printf("🚀 AI ADS Studio Go Backend server listening on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, handlerWithCORS); err != nil {
		log.Printf("Server stopped with error: %v", err)
	}
}
