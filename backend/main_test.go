package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWebDirFindsFrontendDistFromBackendDirectory(t *testing.T) {
	repositoryDir := t.TempDir()
	backendDir := filepath.Join(repositoryDir, "backend")
	frontendDistDir := filepath.Join(repositoryDir, "frontend", "dist")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("create backend directory: %v", err)
	}
	if err := os.MkdirAll(frontendDistDir, 0755); err != nil {
		t.Fatalf("create frontend directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDistDir, "index.html"), []byte("app"), 0600); err != nil {
		t.Fatalf("create frontend entry point: %v", err)
	}

	got, err := resolveWebDir("", filepath.Join(t.TempDir(), "go-build", "app.exe"), backendDir)
	if err != nil {
		t.Fatalf("resolveWebDir() error = %v", err)
	}
	if got != frontendDistDir {
		t.Fatalf("resolveWebDir() = %q, want %q", got, frontendDistDir)
	}
}

func TestResolveWebDirPrefersPackagedWebDirectory(t *testing.T) {
	executableDir := t.TempDir()
	packagedWebDir := filepath.Join(executableDir, "web")
	if err := os.MkdirAll(packagedWebDir, 0755); err != nil {
		t.Fatalf("create packaged web directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packagedWebDir, "index.html"), []byte("app"), 0600); err != nil {
		t.Fatalf("create packaged entry point: %v", err)
	}

	got, err := resolveWebDir("", filepath.Join(executableDir, "app.exe"), t.TempDir())
	if err != nil {
		t.Fatalf("resolveWebDir() error = %v", err)
	}
	if got != packagedWebDir {
		t.Fatalf("resolveWebDir() = %q, want %q", got, packagedWebDir)
	}
}

func TestResolveWebDirUsesConfiguredDirectory(t *testing.T) {
	configuredDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(configuredDir, "index.html"), []byte("app"), 0600); err != nil {
		t.Fatalf("create configured frontend entry point: %v", err)
	}

	got, err := resolveWebDir(configuredDir, "", "")
	if err != nil {
		t.Fatalf("resolveWebDir() error = %v", err)
	}
	if got != configuredDir {
		t.Fatalf("resolveWebDir() = %q, want configured directory %q", got, configuredDir)
	}
}

func TestCORSMiddlewareRejectsUntrustedMutatingRequests(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := corsMiddleware(next, map[string]bool{"http://localhost:5173": true})

	request := httptest.NewRequest(http.MethodPost, "/api/upload", nil)
	request.Header.Set("Origin", "https://untrusted.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("untrusted POST status = %d, want %d", response.Code, http.StatusForbidden)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/upload", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("trusted POST status = %d, want %d", response.Code, http.StatusOK)
	}
}
