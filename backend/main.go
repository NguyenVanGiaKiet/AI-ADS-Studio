package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"ai-ads-studio/backend/internal/handler"
	"ai-ads-studio/backend/internal/service"
	"github.com/joho/godotenv"
)

func corsMiddleware(next http.Handler, allowedOrigins map[string]bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		}
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
	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	authSvc, err := service.NewAuthService(adminUsername, adminPassword)
	if err != nil {
		log.Fatalf("Authentication is not configured: %v", err)
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
	h.AuthService = authSvc

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("GET /api/health", h.HealthCheck)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("GET /api/auth/session", h.AuthSession)
	mux.HandleFunc("POST /api/auth/logout", h.Logout)
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

	allowedOrigins := make(map[string]bool)
	for _, origin := range strings.Split(os.Getenv("FRONTEND_ORIGINS"), ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			allowedOrigins[origin] = true
		}
	}
	if len(allowedOrigins) == 0 {
		allowedOrigins["http://localhost:5173"] = true
	}
	handlerWithAuth := h.RequireAuthentication(mux, allowedOrigins)
	handlerWithCORS := corsMiddleware(handlerWithAuth, allowedOrigins)

	log.Printf("🚀 AI ADS Studio Go Backend server listening on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, handlerWithCORS); err != nil {
		log.Printf("Server stopped with error: %v", err)
	}
}
