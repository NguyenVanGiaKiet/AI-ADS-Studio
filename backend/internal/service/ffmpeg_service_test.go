package service

import (
	"math"
	"os"
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
		remixModeStandard,
		SubtitleSettings{},
		0,
	)
	if err != nil {
		t.Fatalf("GenerateRemixVideo() error = %v", err)
	}
	if size == 0 || filepath.Base(outputPath) != filename {
		t.Fatalf("expected non-empty output video, got filename=%q size=%d", filename, size)
	}
	fingerprint, err := service.FingerprintVideo(outputPath)
	if err != nil {
		t.Fatalf("FingerprintVideo() error = %v", err)
	}
	if len(fingerprint.Frames) == 0 || videoSimilarity(fingerprint, fingerprint) != 1 {
		t.Fatalf("expected usable fingerprint and exact-video similarity, got %d sampled frames", len(fingerprint.Frames))
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
		remixModeStandard,
		SubtitleSettings{},
		0,
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

	_, zoomOutputPath, _, err := service.GenerateRemixVideo(
		0,
		"facezoom01234567",
		[]string{redPath, bluePath},
		2,
		"square",
		"off",
		"",
		remixModeProductZoom,
		SubtitleSettings{},
		0,
	)
	if err != nil {
		t.Fatalf("GenerateRemixVideo() in product zoom mode error = %v", err)
	}
	if _, err := os.Stat(zoomOutputPath); err != nil {
		t.Fatalf("product zoom output video was not created: %v", err)
	}

	_, variedOutputPath, _, err := service.GenerateRemixVideo(
		0,
		"variant012345678",
		[]string{redPath, bluePath},
		2,
		"square",
		"off",
		"",
		remixModeStandard,
		SubtitleSettings{},
		1,
	)
	if err != nil {
		t.Fatalf("GenerateRemixVideo() with a varied candidate error = %v", err)
	}
	if _, err := os.Stat(variedOutputPath); err != nil {
		t.Fatalf("varied candidate video was not created: %v", err)
	}

	_, faceFilteredOutputPath, _, err := service.GenerateRemixVideo(
		0,
		"facefilter012345",
		[]string{redPath, bluePath},
		2,
		"square",
		"off",
		"",
		remixModeExcludeFaces,
		SubtitleSettings{},
		0,
	)
	if err != nil {
		t.Fatalf("GenerateRemixVideo() in face exclusion mode error = %v", err)
	}
	if _, err := os.Stat(faceFilteredOutputPath); err != nil {
		t.Fatalf("face-filtered output video was not created: %v", err)
	}

	_, subtitleOutputPath, _, err := service.GenerateRemixVideo(
		0,
		"subtitles0123456",
		[]string{redPath, bluePath},
		2,
		"square",
		"off",
		"",
		remixModeStandard,
		SubtitleSettings{
			Text:     "Xin chào, sản phẩm dành cho bạn.",
			Position: "bottom",
			Style:    "white_yellow",
		},
		0,
	)
	if err != nil {
		t.Fatalf("GenerateRemixVideo() with karaoke subtitles error = %v", err)
	}
	if _, err := os.Stat(subtitleOutputPath); err != nil {
		t.Fatalf("subtitle output video was not created: %v", err)
	}
}

func TestBuildKaraokeASSUsesSelectedPositionAndColors(t *testing.T) {
	tests := []struct {
		name          string
		position      string
		style         string
		wantAlignment string
		wantColors    string
	}{
		{
			name:          "bottom white and yellow",
			position:      "bottom",
			style:         "white_yellow",
			wantAlignment: ",2,90,90,400,1",
			wantColors:    "Style: Karaoke,Arial,64,&H0000FFFF,&H00FFFFFF,",
		},
		{
			name:          "top white and gray",
			position:      "top",
			style:         "white_gray",
			wantAlignment: ",8,90,90,300,1",
			wantColors:    "Style: Karaoke,Arial,64,&H00FFFFFF,&H00808080,",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildKaraokeASS("Xin chào, bạn nhé.", 8, test.position, test.style)
			if err != nil {
				t.Fatalf("buildKaraokeASS() error = %v", err)
			}
			if !strings.Contains(got, test.wantAlignment) {
				t.Errorf("ASS subtitles missing position settings %q", test.wantAlignment)
			}
			if !strings.Contains(got, test.wantColors) {
				t.Errorf("ASS subtitles missing color settings %q", test.wantColors)
			}
			if !strings.Contains(got, "{\\k") {
				t.Error("ASS subtitles do not contain karaoke word timings")
			}
		})
	}
}

func TestVideoSimilarityDetectsExactAndTemporalSimilarity(t *testing.T) {
	first := VideoFingerprint{Digest: [32]byte{1}, Frames: []uint64{0, ^uint64(0), 0x00000000FFFFFFFF, 0xFFFFFFFF00000000}}
	exact := VideoFingerprint{Digest: [32]byte{2}, Frames: []uint64{0, ^uint64(0), 0x00000000FFFFFFFF, 0xFFFFFFFF00000000}}
	if got := videoSimilarity(first, exact); got != 1 {
		t.Fatalf("videoSimilarity(exact) = %.2f, want 1", got)
	}
	reordered := VideoFingerprint{Digest: [32]byte{3}, Frames: []uint64{0xFFFFFFFF00000000, 0x00000000FFFFFFFF, ^uint64(0), 0}}
	if got := videoSimilarity(first, reordered); got >= 0.6 {
		t.Fatalf("videoSimilarity(reordered) = %.2f, want less than 0.6", got)
	}
	digestDuplicate := VideoFingerprint{Digest: first.Digest}
	if got := videoSimilarity(digestDuplicate, first); got != 1 {
		t.Fatalf("videoSimilarity(same exact digest) = %.2f, want 1", got)
	}
}

func TestGetFaceFreeSegmentsKeepsVideoWithoutFaces(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}
	python := os.Getenv("OPENCV_PYTHON")
	if python == "" {
		python = os.Getenv("PIPER_PYTHON")
	}
	if python == "" {
		python = "python"
	}
	if err := exec.Command(python, "-c", "import cv2").Run(); err != nil {
		t.Skip("OpenCV is not installed for the Python interpreter")
	}

	videoPath := filepath.Join(t.TempDir(), "no_faces.mp4")
	createColorClip(t, ffmpeg, videoPath, "blue")
	service := NewFFmpegService(t.TempDir())
	segments, err := service.getFaceFreeSegments(videoPath)
	if err != nil {
		t.Fatalf("getFaceFreeSegments() error = %v", err)
	}
	if len(segments) != 1 {
		t.Fatalf("getFaceFreeSegments() returned %d segments, want one full-length safe segment", len(segments))
	}
	if math.Abs(segments[0].Start) > 0.01 || math.Abs(segments[0].Duration-2) > 0.1 {
		t.Fatalf("safe segment = %+v, want start 0 and duration about 2 seconds", segments[0])
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
