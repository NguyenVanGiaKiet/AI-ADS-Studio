package service

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type FFmpegService struct {
	HasFFmpeg bool
	OutputDir string
}

func NewFFmpegService(outputDir string) *FFmpegService {
	_, err := exec.LookPath("ffmpeg")
	hasFFmpeg := err == nil
	if hasFFmpeg {
		log.Println("[FFmpegService] FFmpeg binary found on system.")
	} else {
		log.Println("[FFmpegService] FFmpeg binary not found on system. Using fallback mode.")
	}
	os.MkdirAll(outputDir, 0755)
	return &FFmpegService{
		HasFFmpeg: hasFFmpeg,
		OutputDir: outputDir,
	}
}

// GenerateRemixVideo builds a single output video from input source videos.
func (f *FFmpegService) GenerateRemixVideo(
	index int,
	taskID string,
	inputPaths []string,
	duration int,
	aspectRatio string,
	deduplication string,
	narrationPath string,
) (string, string, int64, error) {
	outFilename := fmt.Sprintf("remix_%s_%d.mp4", taskID[:8], index+1)
	outPath := filepath.Join(f.OutputDir, outFilename)

	if !f.HasFFmpeg {
		return "", "", 0, fmt.Errorf("FFmpeg chưa được cài; không thể cắt ghép video")
	}
	if len(inputPaths) == 0 {
		return "", "", 0, fmt.Errorf("không có video nguồn để cắt ghép")
	}
	if err := f.runFFmpegRemix(index, inputPaths, outPath, duration, aspectRatio, deduplication, narrationPath); err != nil {
		return "", "", 0, fmt.Errorf("không thể cắt ghép video: %w", err)
	}
	info, err := os.Stat(outPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("không tìm thấy video đầu ra sau khi xử lý: %w", err)
	}
	return outFilename, outPath, info.Size(), nil
}

func (f *FFmpegService) runFFmpegRemix(
	index int,
	inputPaths []string,
	outPath string,
	duration int,
	aspectRatio string,
	deduplication string,
	narrationPath string,
) error {
	// Build video resolution and scale filter based on aspect ratio
	var vfFilters []string
	switch aspectRatio {
	case "square":
		vfFilters = append(vfFilters, "scale=1080:1080:force_original_aspect_ratio=increase,crop=1080:1080")
	case "landscape":
		vfFilters = append(vfFilters, "scale=1920:1080:force_original_aspect_ratio=increase,crop=1920:1080")
	case "portrait", "vertical":
		vfFilters = append(vfFilters, "scale=1080:1920:force_original_aspect_ratio=increase,crop=1080:1920")
	default:
		vfFilters = append(vfFilters, "scale=1080:1920:force_original_aspect_ratio=increase,crop=1080:1920")
	}

	// Deduplication filters (pitch shift, brightness/contrast tweak, flip/hue shift)
	switch deduplication {
	case "light":
		vfFilters = append(vfFilters, "eq=brightness=0.02:contrast=1.02")
	case "medium":
		vfFilters = append(vfFilters, "eq=brightness=0.03:contrast=1.05:saturation=1.05")
	case "strong":
		vfFilters = append(vfFilters, "eq=brightness=0.05:contrast=1.08:saturation=1.1,hue=h=2")
	}

	videoFilter := strings.Join(vfFilters, ",")
	orderedPaths := make([]string, 0, len(inputPaths))
	sourceDurations := make([]float64, 0, len(inputPaths))
	for segmentIndex := range inputPaths {
		sourcePath := inputPaths[(segmentIndex+index)%len(inputPaths)]
		sourceDuration, err := probeDuration(sourcePath)
		if err != nil {
			return fmt.Errorf("không đọc được thời lượng nguồn %q: %w", filepath.Base(sourcePath), err)
		}
		orderedPaths = append(orderedPaths, sourcePath)
		sourceDurations = append(sourceDurations, sourceDuration)
	}
	segmentDurations := allocateClipDurations(sourceDurations, float64(duration))
	selectedPaths := make([]string, 0, len(orderedPaths))
	selectedDurations := make([]float64, 0, len(orderedPaths))
	for index, segmentDuration := range segmentDurations {
		if segmentDuration > 0.001 {
			selectedPaths = append(selectedPaths, orderedPaths[index])
			selectedDurations = append(selectedDurations, segmentDuration)
		}
	}
	if len(selectedPaths) == 0 {
		return fmt.Errorf("các video nguồn không có thời lượng hợp lệ")
	}
	montageDuration := 0.0
	for _, segmentDuration := range selectedDurations {
		montageDuration += segmentDuration
	}

	var filterParts []string
	var concatParts []string

	args := []string{"-y"}
	for segmentIndex, sourcePath := range selectedPaths {
		segmentDuration := selectedDurations[segmentIndex]
		args = append(args, "-i", sourcePath)
		filterParts = append(filterParts, fmt.Sprintf(
			"[%d:v:0]trim=duration=%.6f,setpts=PTS-STARTPTS,%s,fps=30,setsar=1[v%d]",
			segmentIndex, segmentDuration, videoFilter, segmentIndex,
		))
		concatParts = append(concatParts, fmt.Sprintf("[v%d]", segmentIndex))
		if narrationPath == "" {
			audioLabel := fmt.Sprintf("a%d", segmentIndex)
			if hasAudioStream(sourcePath) {
				filterParts = append(filterParts, fmt.Sprintf(
					"[%d:a:0]atrim=duration=%.6f,asetpts=PTS-STARTPTS,aresample=44100,aformat=sample_fmts=fltp:channel_layouts=stereo[%s]",
					segmentIndex, segmentDuration, audioLabel,
				))
			} else {
				filterParts = append(filterParts, fmt.Sprintf(
					"anullsrc=r=44100:cl=stereo,atrim=duration=%.6f[%s]",
					segmentDuration, audioLabel,
				))
			}
			concatParts = append(concatParts, fmt.Sprintf("[%s]", audioLabel))
		}
	}
	concatAudio := narrationPath == ""
	concatFilter := fmt.Sprintf("%sconcat=n=%d:v=1:a=%d[vout]", strings.Join(concatParts, ""), len(selectedPaths), boolToFFmpegInt(concatAudio))
	if concatAudio {
		concatFilter = strings.TrimSuffix(concatFilter, "[vout]") + "[vout][aout]"
	}
	filterParts = append(filterParts, concatFilter)
	if narrationPath != "" {
		narrationIndex := len(inputPaths)
		args = append(args, "-i", narrationPath)
		filterParts = append(filterParts, fmt.Sprintf(
			"[%d:a:0]apad,atrim=duration=%.6f,asetpts=PTS-STARTPTS,aresample=44100,aformat=sample_fmts=fltp:channel_layouts=stereo[aout]",
			narrationIndex, montageDuration,
		))
	}
	args = append(args,
		"-filter_complex", strings.Join(filterParts, ";"),
		"-map", "[vout]", "-map", "[aout]", "-t", fmt.Sprintf("%.6f", montageDuration),
		"-c:v", "libx264", "-preset", "fast", "-pix_fmt", "yuv420p", "-c:a", "aac", "-movflags", "+faststart", outPath,
	)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %v, output: %s", err, string(output))
	}
	return nil
}

func boolToFFmpegInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func allocateClipDurations(sourceDurations []float64, maxDuration float64) []float64 {
	allocations := make([]float64, len(sourceDurations))
	total := 0.0
	for _, duration := range sourceDurations {
		total += duration
	}
	if maxDuration <= 0 || total <= maxDuration {
		copy(allocations, sourceDurations)
		return allocations
	}

	active := make([]int, 0, len(sourceDurations))
	for index, duration := range sourceDurations {
		if duration > 0 {
			active = append(active, index)
		}
	}
	remaining := maxDuration
	for remaining > 0.001 && len(active) > 0 {
		share := remaining / float64(len(active))
		nextActive := make([]int, 0, len(active))
		for _, index := range active {
			available := sourceDurations[index] - allocations[index]
			portion := math.Min(available, share)
			allocations[index] += portion
			remaining -= portion
			if available-portion > 0.001 {
				nextActive = append(nextActive, index)
			}
		}
		if len(nextActive) == len(active) && len(nextActive) > 0 && remaining > 0 {
			break
		}
		active = nextActive
	}
	return allocations
}

func hasAudioStream(path string) bool {
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return false
	}
	cmd := exec.Command(ffprobe, "-v", "error", "-select_streams", "a:0", "-show_entries", "stream=index", "-of", "csv=p=0", path)
	output, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(output)) != ""
}

func (f *FFmpegService) MontageDuration(inputPaths []string, maxDuration int) (float64, error) {
	if len(inputPaths) == 0 {
		return 0, fmt.Errorf("không có video nguồn")
	}

	var total float64
	for _, path := range inputPaths {
		duration, err := probeDuration(path)
		if err != nil {
			return 0, fmt.Errorf("không đọc được thời lượng nguồn %q: %w", filepath.Base(path), err)
		}
		total += duration
		if maxDuration > 0 && total >= float64(maxDuration) {
			return float64(maxDuration), nil
		}
	}
	return total, nil
}

func probeDuration(path string) (float64, error) {
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return 0, fmt.Errorf("ffprobe chưa được cài")
	}
	cmd := exec.Command(ffprobe, "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil || math.IsNaN(duration) || duration <= 0 {
		return 0, fmt.Errorf("thời lượng không hợp lệ")
	}
	return duration, nil
}
