package service

import (
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestGenerateRemixVideoConcatenatesAllInputs(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe is not installed")
	}

	workDir := t.TempDir()
	redPath := filepath.Join(workDir, "red.mp4")
	bluePath := filepath.Join(workDir, "blue.mp4")
	createColorClip(t, ffmpeg, redPath, "red")
	createColorClip(t, ffmpeg, bluePath, "blue")

	service := NewFFmpegService(filepath.Join(workDir, "outputs"))
	filename, outputPath, size, err := service.GenerateRemixVideo(
		0,
		"0123456789abcdef",
		[]string{redPath, bluePath},
		8,
		"square",
		"off",
		"",
	)
	if err != nil {
		t.Fatalf("GenerateRemixVideo() error = %v", err)
	}
	if size == 0 || filepath.Base(outputPath) != filename {
		t.Fatalf("expected non-empty output video, got filename=%q size=%d", filename, size)
	}

	durationOutput, err := exec.Command(ffprobe, "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", outputPath).Output()
	if err != nil {
		t.Fatalf("ffprobe output duration: %v", err)
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(durationOutput)), 64)
	if err != nil || math.Abs(duration-4) > 0.25 {
		t.Fatalf("output duration = %q, want approximately 4 seconds when sources total 4 and limit is 8", strings.TrimSpace(string(durationOutput)))
	}

	redFrame := extractRGBPixel(t, ffmpeg, outputPath, "0.5")
	blueFrame := extractRGBPixel(t, ffmpeg, outputPath, "2.5")
	if redFrame[0] <= redFrame[2] {
		t.Fatalf("first segment is not red-source footage: RGB=%v", redFrame)
	}
	if blueFrame[2] <= blueFrame[0] {
		t.Fatalf("second segment is not blue-source footage: RGB=%v", blueFrame)
	}

	_, shortOutputPath, _, err := service.GenerateRemixVideo(
		0,
		"abcdef0123456789",
		[]string{redPath, bluePath},
		3,
		"square",
		"off",
		"",
	)
	if err != nil {
		t.Fatalf("GenerateRemixVideo() with shorter limit error = %v", err)
	}
	shortDurationOutput, err := exec.Command(ffprobe, "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", shortOutputPath).Output()
	if err != nil {
		t.Fatalf("ffprobe shorter output duration: %v", err)
	}
	shortDuration, err := strconv.ParseFloat(strings.TrimSpace(string(shortDurationOutput)), 64)
	if err != nil || math.Abs(shortDuration-3) > 0.25 {
		t.Fatalf("short output duration = %q, want approximately 3 seconds", strings.TrimSpace(string(shortDurationOutput)))
	}
	redFrame = extractRGBPixel(t, ffmpeg, shortOutputPath, "0.5")
	blueFrame = extractRGBPixel(t, ffmpeg, shortOutputPath, "2.0")
	if redFrame[0] <= redFrame[2] || blueFrame[2] <= blueFrame[0] {
		t.Fatalf("short output should include a slice of each source: red RGB=%v, blue RGB=%v", redFrame, blueFrame)
	}
}

func createColorClip(t *testing.T, ffmpeg, path, color string) {
	t.Helper()
	cmd := exec.Command(ffmpeg,
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "color=c="+color+":s=320x240:r=24",
		"-t", "2", "-an", "-c:v", "libx264", "-pix_fmt", "yuv420p", path,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create %s clip: %v: %s", color, err, output)
	}
}

func extractRGBPixel(t *testing.T, ffmpeg, path, timestamp string) []byte {
	t.Helper()
	cmd := exec.Command(ffmpeg,
		"-hide_banner", "-loglevel", "error", "-ss", timestamp, "-i", path,
		"-frames:v", "1", "-vf", "scale=1:1:flags=area", "-pix_fmt", "rgb24", "-f", "rawvideo", "pipe:1",
	)
	pixel, err := cmd.Output()
	if err != nil {
		t.Fatalf("extract frame at %ss: %v", timestamp, err)
	}
	if len(pixel) != 3 {
		t.Fatalf("decoded pixel size = %d, want 3 RGB bytes", len(pixel))
	}
	return pixel
}
