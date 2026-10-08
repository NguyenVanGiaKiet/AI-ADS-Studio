package handler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ai-ads-studio/backend/internal/model"
	"ai-ads-studio/backend/internal/service"
)

func TestDownloadAllOutputs(t *testing.T) {
	outputDir := t.TempDir()
	videoName := "remix-output.mp4"
	videoContent := []byte("video data")
	if err := os.WriteFile(filepath.Join(outputDir, videoName), videoContent, 0600); err != nil {
		t.Fatalf("write video fixture: %v", err)
	}
	manifest, err := json.Marshal([]model.OutputVideo{{
		ID:       "video-1",
		Filename: videoName,
	}})
	if err != nil {
		t.Fatalf("marshal video manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "videos.json"), manifest, 0600); err != nil {
		t.Fatalf("write video manifest: %v", err)
	}

	remixService := service.NewRemixService(t.TempDir(), outputDir, nil, nil)
	handler := NewHandler(remixService, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/videos/download", nil)
	response := httptest.NewRecorder()

	handler.DownloadAllOutputs(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("DownloadAllOutputs() status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/zip" {
		t.Fatalf("Content-Type = %q, want application/zip", got)
	}
	if got := response.Header().Get("Content-Disposition"); !strings.Contains(got, `filename=all-videos.zip`) {
		t.Fatalf("Content-Disposition = %q, want all-videos.zip attachment", got)
	}

	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatalf("read downloaded ZIP: %v", err)
	}
	if len(archive.File) != 1 || archive.File[0].Name != videoName {
		t.Fatalf("ZIP entries = %#v, want %q", archive.File, videoName)
	}
	entry, err := archive.File[0].Open()
	if err != nil {
		t.Fatalf("open ZIP entry: %v", err)
	}
	defer entry.Close()
	gotContent, err := io.ReadAll(entry)
	if err != nil {
		t.Fatalf("read ZIP entry: %v", err)
	}
	if !bytes.Equal(gotContent, videoContent) {
		t.Fatalf("ZIP content = %q, want %q", gotContent, videoContent)
	}
}

func TestDownloadAllOutputsReturnsNotFoundWhenEmpty(t *testing.T) {
	remixService := service.NewRemixService(t.TempDir(), t.TempDir(), nil, nil)
	handler := NewHandler(remixService, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/videos/download", nil)
	response := httptest.NewRecorder()

	handler.DownloadAllOutputs(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("DownloadAllOutputs() status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestDownloadSelectedOutputsReturnsOnlyRequestedVideo(t *testing.T) {
	outputDir := t.TempDir()
	outputs := []model.OutputVideo{
		{ID: "video-1", Filename: "remix-1.mp4"},
		{ID: "video-2", Filename: "remix-2.mp4"},
	}
	for _, output := range outputs {
		if err := os.WriteFile(filepath.Join(outputDir, output.Filename), []byte(output.ID), 0600); err != nil {
			t.Fatalf("write video fixture %q: %v", output.Filename, err)
		}
	}
	manifest, err := json.Marshal(outputs)
	if err != nil {
		t.Fatalf("marshal video manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "videos.json"), manifest, 0600); err != nil {
		t.Fatalf("write video manifest: %v", err)
	}

	remixService := service.NewRemixService(t.TempDir(), outputDir, nil, nil)
	handler := NewHandler(remixService, nil)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/videos/download",
		strings.NewReader(`{"videoIds":["video-2"]}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.DownloadSelectedOutputs(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("DownloadSelectedOutputs() status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatalf("read downloaded ZIP: %v", err)
	}
	if len(archive.File) != 1 || archive.File[0].Name != "remix-2.mp4" {
		t.Fatalf("ZIP entries = %#v, want only remix-2.mp4", archive.File)
	}
}
