package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
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
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if origin != "" && !allowedOrigins[origin] {
				http.Error(w, "Origin không được phép", http.StatusForbidden)
				return
			}
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
	listenAddress := os.Getenv("LISTEN_ADDR")
	if listenAddress == "" {
		listenAddress = "127.0.0.1:" + port
	}
	webDir := strings.TrimSpace(os.Getenv("APP_WEB_DIR"))
	if webDir == "" {
		executablePath, err := os.Executable()
		if err == nil {
			webDir = filepath.Join(filepath.Dir(executablePath), "web")
		} else {
			log.Printf("Could not locate application executable: %v", err)
		}
	}
	if webDir == "" {
		webDir = filepath.Join("frontend", "dist")
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
	if _, err := os.Stat(filepath.Join(webDir, "index.html")); err == nil {
		staticFiles := http.FileServer(http.Dir(webDir))
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				http.NotFound(w, r)
				return
			}
			if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/storage/") {
				http.NotFound(w, r)
				return
			}
			requestedPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
			if fs.ValidPath(requestedPath) {
				requestedFile := filepath.Join(webDir, filepath.FromSlash(requestedPath))
				if info, err := os.Stat(requestedFile); err == nil && !info.IsDir() {
					staticFiles.ServeHTTP(w, r)
					return
				}
			}
			http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
		}))
	} else if !os.IsNotExist(err) {
		log.Fatalf("Could not access frontend files in %q: %v", webDir, err)
	} else {
		log.Printf("Frontend files not found in %q; serving API only", webDir)
	}

	allowedOrigins := make(map[string]bool)
	for _, origin := range strings.Split(os.Getenv("FRONTEND_ORIGINS"), ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			allowedOrigins[origin] = true
		}
	}
	if len(allowedOrigins) == 0 {
		allowedOrigins["http://localhost:"+port] = true
		allowedOrigins["http://127.0.0.1:"+port] = true
		allowedOrigins["http://localhost:5173"] = true
	}
	handlerWithCORS := corsMiddleware(mux, allowedOrigins)

	log.Printf("🚀 AI ADS Studio server listening on %s\n", listenAddress)
	if err := http.ListenAndServe(listenAddress, handlerWithCORS); err != nil {
		log.Printf("Server stopped with error: %v", err)
	}
}
