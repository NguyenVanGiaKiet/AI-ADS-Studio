package service

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"ai-ads-studio/backend/internal/model"
)

func TestWriteTaskOutputsZip(t *testing.T) {
	outputDir := t.TempDir()
	videoContent := []byte("fake video content")
	filename := "remix_task_1.mp4"
	if err := os.WriteFile(filepath.Join(outputDir, filename), videoContent, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	svc := NewRemixService(t.TempDir(), outputDir, nil, nil)
	svc.outputs["video-1"] = model.OutputVideo{
		ID:       "video-1",
		TaskID:   "task-1",
		Filename: filename,
	}

	var archiveBytes bytes.Buffer
	if err := svc.WriteTaskOutputsZip("task-1", &archiveBytes); err != nil {
		t.Fatalf("WriteTaskOutputsZip() error = %v", err)
	}

	archive, err := zip.NewReader(bytes.NewReader(archiveBytes.Bytes()), int64(archiveBytes.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}
	if len(archive.File) != 1 || archive.File[0].Name != filename {
		t.Fatalf("ZIP entries = %#v, want one entry named %q", archive.File, filename)
	}
	entry, err := archive.File[0].Open()
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer entry.Close()
	got, err := io.ReadAll(entry)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !bytes.Equal(got, videoContent) {
		t.Fatalf("ZIP entry contents = %q, want %q", got, videoContent)
	}
}

func TestWriteTaskOutputsZipReturnsNotFound(t *testing.T) {
	svc := NewRemixService(t.TempDir(), t.TempDir(), nil, nil)
	if err := svc.WriteTaskOutputsZip("missing-task", &bytes.Buffer{}); !errors.Is(err, ErrTaskOutputsNotFound) {
		t.Fatalf("WriteTaskOutputsZip() error = %v, want ErrTaskOutputsNotFound", err)
	}
}

func TestIsDuplicateScriptIgnoresPunctuationAndCase(t *testing.T) {
	previous := []string{"Xin chào, sản phẩm tuyệt vời!"}
	if !isDuplicateScript("XIN CHÀO sản phẩm tuyệt vời.", previous) {
		t.Fatal("isDuplicateScript() = false, want punctuation/case-insensitive duplicate")
	}
	if isDuplicateScript("Khám phá sản phẩm mới ngay hôm nay.", previous) {
		t.Fatal("isDuplicateScript() = true, want distinct script")
	}
}
