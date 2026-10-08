package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	mathbits "math/bits"
	mathrand "math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	remixModeStandard     = "standard"
	remixModeExcludeFaces = "exclude_faces"
	remixModeProductZoom  = "product_zoom"
)

type FFmpegService struct {
	HasFFmpeg        bool
	OutputDir        string
	cacheMu          sync.RWMutex
	faceCache        map[string]cachedFaceSegments
	fingerprintCache map[string]cachedVideoFingerprint
}

type videoSegment struct {
	Start    float64 `json:"start"`
	Duration float64 `json:"duration"`
}

type videoSegmentWithPath struct {
	Path     string
	Start    float64
	Duration float64
}

type cachedFaceSegments struct {
	Size         int64
	ModifiedAt   time.Time
	SafeSegments []videoSegment
}

type VideoFingerprint struct {
	Digest [32]byte
	Frames []uint64
}

type cachedVideoFingerprint struct {
	Size       int64
	ModifiedAt time.Time
	Value      VideoFingerprint
}

type SubtitleSettings struct {
	Text     string
	Position string
	Style    string
}

type alignedSubtitleWord struct {
	start float64
	end   float64
	word  string
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
		HasFFmpeg:        hasFFmpeg,
		OutputDir:        outputDir,
		faceCache:        make(map[string]cachedFaceSegments),
		fingerprintCache: make(map[string]cachedVideoFingerprint),
	}
}

// GenerateRemixVideo builds a single output video from input source videos.
func (f *FFmpegService) GenerateRemixVideo(
	index int,
	taskID string,
	inputPaths []string,
	duration int,
	cutSensitivity string,
	deduplication string,
	narrationPath string,
	remixMode string,
	subtitles SubtitleSettings,
	variant int,
) (string, string, int64, error) {
	outFilename := fmt.Sprintf("remix_%s_%d.mp4", taskID[:8], index+1)
	if variant > 0 {
		outFilename = fmt.Sprintf("remix_%s_%d_candidate%d.mp4", taskID[:8], index+1, variant)
	}
	outPath := filepath.Join(f.OutputDir, outFilename)

	if !f.HasFFmpeg {
		return "", "", 0, fmt.Errorf("FFmpeg chưa được cài; không thể cắt ghép video")
	}
	if len(inputPaths) == 0 {
		return "", "", 0, fmt.Errorf("không có video nguồn để cắt ghép")
	}
	if err := f.runFFmpegRemix(index, inputPaths, outPath, duration, cutSensitivity, deduplication, narrationPath, remixMode, subtitles, variant); err != nil {
		_ = os.Remove(outPath)
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
	cutSensitivity string,
	deduplication string,
	narrationPath string,
	remixMode string,
	subtitles SubtitleSettings,
	variant int,
) error {
	// Build video resolution and scale filter (defaults to vertical 9:16)
	var vfFilters []string
	if remixMode == remixModeProductZoom {
		vfFilters = append(vfFilters, "crop=iw*0.70:ih*0.70:(iw-ow)/2:ih-oh")
	}
	vfFilters = append(vfFilters, "scale=1080:1920:force_original_aspect_ratio=increase,crop=1080:1920")

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
	var sourceSegments []videoSegmentWithPath
	for segmentIndex := range inputPaths {
		sourcePath := inputPaths[(segmentIndex+index)%len(inputPaths)]
		sourceDuration, err := probeDuration(sourcePath)
		if err != nil {
			return fmt.Errorf("không đọc được thời lượng nguồn %q: %w", filepath.Base(sourcePath), err)
		}
		if remixMode == remixModeExcludeFaces {
			safeSegments, err := f.getFaceFreeSegments(sourcePath)
			if err != nil {
				return fmt.Errorf("không thể nhận diện khuôn mặt trong %q: %w", filepath.Base(sourcePath), err)
			}
			for _, safeSegment := range safeSegments {
				sourceSegments = append(sourceSegments, videoSegmentWithPath{
					Path:     sourcePath,
					Start:    safeSegment.Start,
					Duration: safeSegment.Duration,
				})
			}
		} else {
			sourceSegments = append(sourceSegments, videoSegmentWithPath{
				Path:     sourcePath,
				Duration: sourceDuration,
			})
		}
	}
	var selectedSegments []videoSegmentWithPath
	if variant > 0 {
		selectedSegments = allocateVariantSegments(sourceSegments, float64(duration), mathrand.New(mathrand.NewSource(time.Now().UnixNano()+int64(variant)*7919+int64(index)*104729)))
	} else {
		selectedSegments = allocateSourceSegments(sourceSegments, float64(duration))
	}
	if len(selectedSegments) == 0 {
		return fmt.Errorf("không còn cảnh phù hợp để tạo video; hãy dùng video dài hơn hoặc chọn chế độ khác")
	}
	montageDuration := 0.0
	for _, segment := range selectedSegments {
		montageDuration += segment.Duration
	}

	var filterParts []string
	var concatParts []string

	args := []string{"-y"}
	for segmentIndex, segment := range selectedSegments {
		args = append(args, "-ss", fmt.Sprintf("%.6f", segment.Start), "-i", segment.Path)
		filterParts = append(filterParts, fmt.Sprintf(
			"[%d:v:0]trim=duration=%.6f,setpts=PTS-STARTPTS,%s,fps=30,setsar=1[v%d]",
			segmentIndex, segment.Duration, videoFilter, segmentIndex,
		))
		concatParts = append(concatParts, fmt.Sprintf("[v%d]", segmentIndex))
		if narrationPath == "" {
			audioLabel := fmt.Sprintf("a%d", segmentIndex)
			if hasAudioStream(segment.Path) {
				filterParts = append(filterParts, fmt.Sprintf(
					"[%d:a:0]atrim=duration=%.6f,asetpts=PTS-STARTPTS,aresample=44100,aformat=sample_fmts=fltp:channel_layouts=stereo[%s]",
					segmentIndex, segment.Duration, audioLabel,
				))
			} else {
				filterParts = append(filterParts, fmt.Sprintf(
					"anullsrc=r=44100:cl=stereo,atrim=duration=%.6f[%s]",
					segment.Duration, audioLabel,
				))
			}
			concatParts = append(concatParts, fmt.Sprintf("[%s]", audioLabel))
		}
	}
	concatAudio := narrationPath == ""
	concatFilter := fmt.Sprintf("%sconcat=n=%d:v=1:a=%d[vout]", strings.Join(concatParts, ""), len(selectedSegments), boolToFFmpegInt(concatAudio))
	if concatAudio {
		concatFilter = strings.TrimSuffix(concatFilter, "[vout]") + "[vout][aout]"
	}
	filterParts = append(filterParts, concatFilter)
	videoOutput := "[vout]"
	if strings.TrimSpace(subtitles.Text) != "" {
		subtitlePath, err := f.writeKaraokeSubtitles(outPath, montageDuration, subtitles)
		if err != nil {
			return err
		}
		defer os.Remove(subtitlePath)
		filterParts = append(filterParts, fmt.Sprintf(
			"[vout]subtitles=filename='%s'[vsub]",
			escapeSubtitleFilterPath(subtitlePath),
		))
		videoOutput = "[vsub]"
	}
	if narrationPath != "" {
		narrationIndex := len(selectedSegments)
		args = append(args, "-i", narrationPath)
		filterParts = append(filterParts, fmt.Sprintf(
			"[%d:a:0]apad,atrim=duration=%.6f,asetpts=PTS-STARTPTS,aresample=44100,aformat=sample_fmts=fltp:channel_layouts=stereo[aout]",
			narrationIndex, montageDuration,
		))
	}
	args = append(args,
		"-filter_complex", strings.Join(filterParts, ";"),
		"-map", videoOutput, "-map", "[aout]", "-t", fmt.Sprintf("%.6f", montageDuration),
		"-c:v", "libx264", "-preset", "fast", "-pix_fmt", "yuv420p", "-c:a", "aac", "-movflags", "+faststart", outPath,
	)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %v, output: %s", err, string(output))
	}
	return nil
}

func (f *FFmpegService) writeKaraokeSubtitles(videoPath string, duration float64, settings SubtitleSettings) (string, error) {
	assContent, err := buildKaraokeASS(settings.Text, duration, settings.Position, settings.Style)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(f.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("không thể tạo thư mục lưu phụ đề: %w", err)
	}
	file, err := os.CreateTemp(f.OutputDir, "captions-*.ass")
	if err != nil {
		return "", fmt.Errorf("không thể tạo tệp phụ đề cho %q: %w", filepath.Base(videoPath), err)
	}
	path := file.Name()
	if _, err := file.WriteString(assContent); err != nil {
		file.Close()
		os.Remove(path)
		return "", fmt.Errorf("không thể ghi tệp phụ đề: %w", err)
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("không thể hoàn tất tệp phụ đề: %w", err)
	}
	return path, nil
}

func buildKaraokeASS(text string, duration float64, position, style string) (string, error) {
	if duration <= 0 || math.IsNaN(duration) || math.IsInf(duration, 0) {
		return "", fmt.Errorf("thời lượng phụ đề không hợp lệ")
	}
	alignment := 2
	marginV := 400
	if position == "top" {
		alignment = 8
		marginV = 300
	} else if position != "" && position != "bottom" {
		return "", fmt.Errorf("vị trí phụ đề không hợp lệ: %q", position)
	}

	primaryColor := "&H0000FFFF"
	secondaryColor := "&H00FFFFFF"
	if style == "white_gray" {
		primaryColor = "&H00FFFFFF"
		secondaryColor = "&H00808080"
	} else if style != "" && style != "white_yellow" {
		return "", fmt.Errorf("kiểu màu phụ đề không hợp lệ: %q", style)
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return "", fmt.Errorf("không có nội dung để tạo phụ đề")
	}
	alignedWords := alignSubtitleTimings(words, duration)
	const playResX, playResY = 1080, 1920
	const centisecondsPerSecond = 100
	totalCentiseconds := int(math.Round(duration * centisecondsPerSecond))
	if totalCentiseconds < len(words) {
		return "", fmt.Errorf("thời lượng quá ngắn để căn phụ đề cho toàn bộ kịch bản")
	}

	var builder strings.Builder
	builder.WriteString("[Script Info]\n")
	builder.WriteString("ScriptType: v4.00+\n")
	builder.WriteString("WrapStyle: 2\n")
	builder.WriteString("ScaledBorderAndShadow: yes\n")
	builder.WriteString(fmt.Sprintf("PlayResX: %d\nPlayResY: %d\n\n", playResX, playResY))
	builder.WriteString("[V4+ Styles]\n")
	builder.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	builder.WriteString(fmt.Sprintf("Style: Karaoke,Arial,64,%s,%s,&H00000000,&H64000000,-1,0,0,0,100,100,0,0,1,4,1,%d,90,90,%d,1\n\n", primaryColor, secondaryColor, alignment, marginV))
	builder.WriteString("[Events]\n")
	builder.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")

	for groupStart := 0; groupStart < len(words); {
		groupEnd := groupStart + 1
		for groupEnd < len(words) && groupEnd-groupStart < 6 && !endsSubtitleSentence(words[groupEnd-1]) {
			groupEnd++
		}
		var line strings.Builder
		groupStartTime := int(math.Round(alignedWords[groupStart].start * centisecondsPerSecond))
		for wordIndex := groupStart; wordIndex < groupEnd; wordIndex++ {
			startCentiseconds := int(math.Round(alignedWords[wordIndex].start * centisecondsPerSecond))
			endCentiseconds := int(math.Round(alignedWords[wordIndex].end * centisecondsPerSecond))
			if endCentiseconds <= startCentiseconds {
				endCentiseconds = startCentiseconds + 1
			}
			line.WriteString(fmt.Sprintf(
				"{\\k%d}%s ",
				endCentiseconds-startCentiseconds,
				escapeASSText(words[wordIndex]),
			))
		}
		builder.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Karaoke,,0,0,0,,%s\n",
			formatASSTime(groupStartTime),
			formatASSTime(int(math.Round(alignedWords[groupEnd-1].end*centisecondsPerSecond))),
			strings.TrimSpace(line.String()),
		))
		groupStart = groupEnd
	}
	return builder.String(), nil
}

func alignSubtitleTimings(words []string, duration float64) []alignedSubtitleWord {
	weights := make([]float64, len(words))
	totalWeight := 0.0
	for index, word := range words {
		weight := float64(utf8.RuneCountInString(word))
		if strings.ContainsAny(word, ",;:") {
			weight += 3
		}
		if strings.ContainsAny(word, ".!?。！？") {
			weight += 8
		}
		weights[index] = weight
		totalWeight += weight
	}
	aligned := make([]alignedSubtitleWord, len(words))
	elapsedWeight := 0.0
	for index, word := range words {
		start := duration * elapsedWeight / totalWeight
		elapsedWeight += weights[index]
		end := duration * elapsedWeight / totalWeight
		aligned[index] = alignedSubtitleWord{start: start, end: end, word: word}
	}
	return aligned
}

func endsSubtitleSentence(word string) bool {
	return strings.ContainsAny(word, ".!?。！？")
}

func escapeASSText(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "{", "\\{")
	return strings.ReplaceAll(text, "}", "\\}")
}

func formatASSTime(centiseconds int) string {
	hours := centiseconds / 360000
	centiseconds %= 360000
	minutes := centiseconds / 6000
	centiseconds %= 6000
	seconds := centiseconds / 100
	centiseconds %= 100
	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, seconds, centiseconds)
}

func escapeSubtitleFilterPath(path string) string {
	path = filepath.ToSlash(path)
	path = strings.ReplaceAll(path, "'", "\\'")
	return strings.ReplaceAll(path, ":", "\\:")
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

func allocateSourceSegments(segments []videoSegmentWithPath, maxDuration float64) []videoSegmentWithPath {
	durations := make([]float64, len(segments))
	for index, segment := range segments {
		durations[index] = segment.Duration
	}
	allocations := allocateClipDurations(durations, maxDuration)
	selected := make([]videoSegmentWithPath, 0, len(segments))
	for index, allocated := range allocations {
		if allocated <= 0.001 {
			continue
		}
		segment := segments[index]
		segment.Duration = allocated
		selected = append(selected, segment)
	}
	return selected
}

func allocateVariantSegments(segments []videoSegmentWithPath, maxDuration float64, random *mathrand.Rand) []videoSegmentWithPath {
	if len(segments) == 0 {
		return nil
	}
	shuffled := append([]videoSegmentWithPath(nil), segments...)
	random.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	durations := make([]float64, len(shuffled))
	for index, segment := range shuffled {
		durations[index] = segment.Duration
	}
	allocations := allocateClipDurations(durations, maxDuration)
	selected := make([]videoSegmentWithPath, 0, len(shuffled))
	for index, allocated := range allocations {
		if allocated <= 0.001 {
			continue
		}
		segment := shuffled[index]
		headroom := segment.Duration - allocated
		if headroom > 0.5 {
			segment.Start += random.Float64() * headroom
		}
		segment.Duration = allocated
		selected = append(selected, segment)
	}
	return selected
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

func (f *FFmpegService) MontageDurationForMode(inputPaths []string, maxDuration int, remixMode string) (float64, error) {
	if remixMode != remixModeExcludeFaces {
		return f.MontageDuration(inputPaths, maxDuration)
	}
	if len(inputPaths) == 0 {
		return 0, fmt.Errorf("không có video nguồn")
	}
	var total float64
	for _, path := range inputPaths {
		segments, err := f.getFaceFreeSegments(path)
		if err != nil {
			return 0, fmt.Errorf("không thể nhận diện khuôn mặt trong %q: %w", filepath.Base(path), err)
		}
		for _, segment := range segments {
			total += segment.Duration
		}
		if maxDuration > 0 && total >= float64(maxDuration) {
			return float64(maxDuration), nil
		}
	}
	if total <= 0 {
		return 0, fmt.Errorf("không tìm thấy đủ cảnh không có khuôn mặt; hãy dùng video khác hoặc chọn chế độ remix khác")
	}
	return total, nil
}

func (f *FFmpegService) getFaceFreeSegments(videoPath string) ([]videoSegment, error) {
	info, err := os.Stat(videoPath)
	if err != nil {
		return nil, fmt.Errorf("không thể đọc video: %w", err)
	}
	f.cacheMu.RLock()
	cached, ok := f.faceCache[videoPath]
	f.cacheMu.RUnlock()
	if ok && cached.Size == info.Size() && cached.ModifiedAt.Equal(info.ModTime()) {
		return append([]videoSegment(nil), cached.SafeSegments...), nil
	}

	scriptPath, err := findFaceDetectorScript()
	if err != nil {
		return nil, err
	}
	python := strings.TrimSpace(os.Getenv("OPENCV_PYTHON"))
	if python == "" {
		python = strings.TrimSpace(os.Getenv("PIPER_PYTHON"))
	}
	if python == "" {
		python = "python"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, scriptPath, videoPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("nhận diện khuôn mặt chạy quá thời gian cho phép: %w", ctx.Err())
		}
		return nil, fmt.Errorf("không thể chạy bộ nhận diện OpenCV (hãy cài opencv-python-headless vào Python backend): %w: %s", err, strings.TrimSpace(string(output)))
	}
	var result struct {
		SafeSegments []videoSegment `json:"safeSegments"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("kết quả nhận diện khuôn mặt không hợp lệ: %w", err)
	}
	for _, segment := range result.SafeSegments {
		if math.IsNaN(segment.Start) || math.IsInf(segment.Start, 0) || math.IsNaN(segment.Duration) || math.IsInf(segment.Duration, 0) || segment.Start < 0 || segment.Duration <= 0 {
			return nil, fmt.Errorf("kết quả nhận diện khuôn mặt chứa mốc thời gian không hợp lệ")
		}
	}

	f.cacheMu.Lock()
	f.faceCache[videoPath] = cachedFaceSegments{
		Size:         info.Size(),
		ModifiedAt:   info.ModTime(),
		SafeSegments: append([]videoSegment(nil), result.SafeSegments...),
	}
	f.cacheMu.Unlock()
	return result.SafeSegments, nil
}

func findFaceDetectorScript() (string, error) {
	candidates := []string{
		filepath.Join("scripts", "detect_faces.py"),
		filepath.Join("backend", "scripts", "detect_faces.py"),
	}
	if _, sourcePath, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(sourcePath), "..", "..", "scripts", "detect_faces.py"))
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			if absolutePath, err := filepath.Abs(candidate); err == nil {
				return absolutePath, nil
			}
			return candidate, nil
		}
	}
	return "", fmt.Errorf("không tìm thấy backend/scripts/detect_faces.py")
}

func (f *FFmpegService) FingerprintVideo(videoPath string) (VideoFingerprint, error) {
	info, err := os.Stat(videoPath)
	if err != nil {
		return VideoFingerprint{}, fmt.Errorf("không thể đọc video để kiểm tra trùng lặp: %w", err)
	}
	f.cacheMu.RLock()
	cached, ok := f.fingerprintCache[videoPath]
	f.cacheMu.RUnlock()
	if ok && cached.Size == info.Size() && cached.ModifiedAt.Equal(info.ModTime()) {
		return cached.Value, nil
	}

	file, err := os.Open(videoPath)
	if err != nil {
		return VideoFingerprint{}, fmt.Errorf("không thể mở video để tạo fingerprint: %w", err)
	}
	digest, err := sha256File(file)
	closeErr := file.Close()
	if err != nil {
		return VideoFingerprint{}, err
	}
	if closeErr != nil {
		return VideoFingerprint{}, fmt.Errorf("không thể đóng video sau khi đọc fingerprint: %w", closeErr)
	}

	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return VideoFingerprint{}, fmt.Errorf("ffmpeg chưa được cài để so sánh độ tương đồng")
	}
	cmd := exec.Command(ffmpeg,
		"-hide_banner", "-loglevel", "error",
		"-i", videoPath,
		"-vf", "fps=1,scale=9:8:flags=area,format=gray",
		"-pix_fmt", "gray", "-f", "rawvideo", "pipe:1",
	)
	rawFrames, err := cmd.Output()
	if err != nil {
		var details string
		if exitError, ok := err.(*exec.ExitError); ok {
			details = strings.TrimSpace(string(exitError.Stderr))
		}
		return VideoFingerprint{}, fmt.Errorf("không thể lấy mẫu khung hình để so sánh video: %w: %s", err, details)
	}
	const frameBytes = 9 * 8
	if len(rawFrames) < frameBytes {
		return VideoFingerprint{}, fmt.Errorf("không đủ khung hình để so sánh video %q", filepath.Base(videoPath))
	}
	fingerprint := VideoFingerprint{Digest: digest}
	for offset := 0; offset+frameBytes <= len(rawFrames); offset += frameBytes {
		frame := rawFrames[offset : offset+frameBytes]
		var hash uint64
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				if frame[y*9+x] > frame[y*9+x+1] {
					hash |= uint64(1) << (y*8 + x)
				}
			}
		}
		fingerprint.Frames = append(fingerprint.Frames, hash)
	}

	f.cacheMu.Lock()
	f.fingerprintCache[videoPath] = cachedVideoFingerprint{
		Size:       info.Size(),
		ModifiedAt: info.ModTime(),
		Value:      fingerprint,
	}
	f.cacheMu.Unlock()
	return fingerprint, nil
}

func sha256File(file *os.File) ([32]byte, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return [32]byte{}, fmt.Errorf("không thể đọc video để tạo checksum: %w", err)
	}
	var digest [32]byte
	copy(digest[:], hash.Sum(nil))
	return digest, nil
}

func videoSimilarity(candidate VideoFingerprint, reference VideoFingerprint) float64 {
	if candidate.Digest != ([32]byte{}) && candidate.Digest == reference.Digest {
		return 1
	}
	if len(candidate.Frames) == 0 || len(reference.Frames) == 0 {
		return 0
	}
	matches := 0
	const maxHashDistance = 10
	for candidateIndex, candidateFrame := range candidate.Frames {
		referenceIndex := candidateIndex * len(reference.Frames) / len(candidate.Frames)
		bestDistance := 64
		for offset := -1; offset <= 1; offset++ {
			nearbyIndex := referenceIndex + offset
			if nearbyIndex < 0 || nearbyIndex >= len(reference.Frames) {
				continue
			}
			referenceFrame := reference.Frames[nearbyIndex]
			if distance := mathbits.OnesCount64(candidateFrame ^ referenceFrame); distance < bestDistance {
				bestDistance = distance
			}
		}
		if bestDistance <= maxHashDistance {
			matches++
		}
	}
	return float64(matches) / float64(len(candidate.Frames))
}

func maxSimilarity(candidate VideoFingerprint, references []VideoFingerprint) float64 {
	best := 0.0
	for _, reference := range references {
		if similarity := videoSimilarity(candidate, reference); similarity > best {
			best = similarity
		}
	}
	return best
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
