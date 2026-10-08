package service

import (
	"archive/zip"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"ai-ads-studio/backend/internal/model"
)

type RemixService struct {
	UploadDir     string
	OutputDir     string
	OutputsFile   string
	FFmpegService *FFmpegService
	TTSService    *TTSService

	mu        sync.RWMutex
	noveltyMu sync.Mutex
	uploads   map[string]model.UploadedVideo
	tasks     map[string]*model.RemixTask
	outputs   map[string]model.OutputVideo
}

var ErrTaskOutputsNotFound = errors.New("không tìm thấy video đầu ra cho tiến trình")

func NewRemixService(uploadDir, outputDir string, ffmpegSvc *FFmpegService, ttsSvc *TTSService) *RemixService {
	os.MkdirAll(uploadDir, 0755)
	os.MkdirAll(outputDir, 0755)

	service := &RemixService{
		UploadDir:     uploadDir,
		OutputDir:     outputDir,
		OutputsFile:   filepath.Join(outputDir, "videos.json"),
		FFmpegService: ffmpegSvc,
		TTSService:    ttsSvc,
		uploads:       make(map[string]model.UploadedVideo),
		tasks:         make(map[string]*model.RemixTask),
		outputs:       make(map[string]model.OutputVideo),
	}
	service.loadOutputs()
	return service
}

func (s *RemixService) loadOutputs() {
	knownFiles := make(map[string]bool)
	if data, err := os.ReadFile(s.OutputsFile); err == nil {
		var outputs []model.OutputVideo
		if err := json.Unmarshal(data, &outputs); err != nil {
			log.Printf("[RemixService] Could not read output manifest: %v", err)
		} else {
			for _, output := range outputs {
				if _, err := os.Stat(filepath.Join(s.OutputDir, output.Filename)); err == nil {
					s.outputs[output.ID] = output
					knownFiles[output.Filename] = true
				}
			}
		}
	} else if !os.IsNotExist(err) {
		log.Printf("[RemixService] Could not open output manifest: %v", err)
	}

	entries, err := os.ReadDir(s.OutputDir)
	if err != nil {
		log.Printf("[RemixService] Could not scan output directory: %v", err)
		return
	}
	for _, entry := range entries {
		filename := entry.Name()
		if entry.IsDir() || knownFiles[filename] || !strings.EqualFold(filepath.Ext(filename), ".mp4") {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.Size() == 0 {
			continue
		}

		stem := strings.TrimSuffix(filename, filepath.Ext(filename))
		parts := strings.Split(stem, "_")
		if len(parts) < 3 || parts[0] != "remix" {
			continue
		}
		sequence := parts[len(parts)-1]
		taskID := strings.Join(parts[1:len(parts)-1], "_")
		output := model.OutputVideo{
			ID:        "recovered-" + stem,
			TaskID:    taskID,
			Title:     fmt.Sprintf("Remix #%s (đã khôi phục)", sequence),
			Filename:  filename,
			URL:       "/storage/outputs/" + filename,
			Duration:  probeVideoDuration(filepath.Join(s.OutputDir, filename)),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		}
		s.outputs[output.ID] = output
	}

	if len(s.outputs) > 0 {
		if err := s.persistOutputs(); err != nil {
			log.Printf("[RemixService] Could not persist recovered outputs: %v", err)
		}
	}
}

func probeVideoDuration(path string) int {
	output, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path).Output()
	if err != nil {
		return 0
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0
	}
	return int(duration + 0.5)
}

func (s *RemixService) persistOutputs() error {
	outputs := make([]model.OutputVideo, 0, len(s.outputs))
	for _, output := range s.outputs {
		outputs = append(outputs, output)
	}
	sort.Slice(outputs, func(i, j int) bool {
		return outputs[i].CreatedAt.Before(outputs[j].CreatedAt)
	})

	data, err := json.MarshalIndent(outputs, "", "  ")
	if err != nil {
		return err
	}
	temporaryFile := s.OutputsFile + ".tmp"
	if err := os.WriteFile(temporaryFile, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(temporaryFile, s.OutputsFile); err != nil {
		_ = os.Remove(temporaryFile)
		return err
	}
	return nil
}

func generateID() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// SaveUpload stores an uploaded multipart file to disk.
func (s *RemixService) SaveUpload(fileHeader *multipart.FileHeader) (*model.UploadedVideo, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	id := generateID()
	ext := filepath.Ext(fileHeader.Filename)
	savedFilename := fmt.Sprintf("%s_%s%s", id, filepath.Base(fileHeader.Filename), ext)
	savedPath := filepath.Join(s.UploadDir, savedFilename)

	out, err := os.Create(savedPath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	size, err := io.Copy(out, file)
	if err != nil {
		return nil, err
	}

	uploaded := model.UploadedVideo{
		ID:        id,
		Filename:  fileHeader.Filename,
		Size:      size,
		Path:      savedPath,
		URL:       "/storage/uploads/" + savedFilename,
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.uploads[id] = uploaded
	s.mu.Unlock()

	return &uploaded, nil
}

// GetUploads returns all uploaded source videos.
func (s *RemixService) GetUploads() []model.UploadedVideo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.UploadedVideo, 0, len(s.uploads))
	for _, v := range s.uploads {
		result = append(result, v)
	}
	return result
}

// CreateTask starts a new video remix job.
func (s *RemixService) CreateTask(req model.RemixRequest) (*model.RemixTask, error) {
	if req.OutputCount <= 0 {
		req.OutputCount = 5
	}
	if req.Duration <= 0 {
		req.Duration = 30
	}
	if req.CutSensitivity == "" {
		req.CutSensitivity = "medium"
	}
	if req.RemixMode == "" {
		req.RemixMode = "standard"
	}
	switch req.RemixMode {
	case "standard", "exclude_faces", "product_zoom":
	default:
		return nil, fmt.Errorf("chế độ remix không hợp lệ: %q", req.RemixMode)
	}
	if req.FollowSubtitles && !req.ReplaceVoice {
		return nil, fmt.Errorf("phụ đề đồng bộ yêu cầu bật giọng đọc thay thế")
	}
	if req.FollowSubtitles {
		if req.SubtitlePosition == "" {
			req.SubtitlePosition = "bottom"
		}
		if req.SubtitleStyle == "" {
			req.SubtitleStyle = "white_yellow"
		}
		if req.SubtitlePosition != "bottom" && req.SubtitlePosition != "top" {
			return nil, fmt.Errorf("vị trí phụ đề không hợp lệ: %q", req.SubtitlePosition)
		}
		if req.SubtitleStyle != "white_yellow" && req.SubtitleStyle != "white_gray" {
			return nil, fmt.Errorf("kiểu màu phụ đề không hợp lệ: %q", req.SubtitleStyle)
		}
	}

	taskID := generateID()
	task := &model.RemixTask{
		ID:           taskID,
		Status:       "processing",
		Progress:     0,
		Message:      "Khởi tạo tiến trình remix...",
		Request:      req,
		OutputVideos: []model.OutputVideo{},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	s.mu.Lock()
	s.tasks[taskID] = task
	s.mu.Unlock()

	// Launch background processing worker
	go s.processRemixTask(taskID)

	return task, nil
}

func (s *RemixService) processRemixTask(taskID string) {
	s.mu.RLock()
	task, exists := s.tasks[taskID]
	s.mu.RUnlock()

	if !exists {
		return
	}

	// Update progress: analyzing source videos
	s.updateTaskStatus(taskID, "processing", 10, "Đang phân tích các video nguồn...")
	time.Sleep(500 * time.Millisecond)

	// Resolve input paths
	s.mu.RLock()
	var inputPaths []string
	for _, vid := range task.Request.VideoIDs {
		if u, ok := s.uploads[vid]; ok {
			inputPaths = append(inputPaths, u.Path)
		}
	}
	// Fallback to all uploads if specific IDs not found
	if len(inputPaths) == 0 {
		for _, u := range s.uploads {
			inputPaths = append(inputPaths, u.Path)
		}
	}
	s.mu.RUnlock()

	montageDuration := float64(task.Request.Duration)
	if task.Request.ReplaceVoice {
		if s.TTSService == nil {
			s.updateTaskStatus(taskID, "failed", 20, "Dịch vụ TTS chưa được khởi tạo.")
			return
		}
		var err error
		montageDuration, err = s.FFmpegService.MontageDurationForMode(inputPaths, task.Request.Duration, task.Request.RemixMode)
		if err != nil {
			s.updateTaskStatus(taskID, "failed", 20, fmt.Sprintf("Không thể xác định thời lượng video ghép: %v", err))
			return
		}
	}

	totalOutputs := task.Request.OutputCount
	var generatedVideos []model.OutputVideo
	var scripts []string
	var similarFallbackCount int
	references := make([]VideoFingerprint, 0)
	const maxNoveltyAttempts = 5
	const similarityThreshold = 0.6

	s.noveltyMu.Lock()
	defer s.noveltyMu.Unlock()
	s.mu.RLock()
	referencePaths := make([]string, 0, len(s.outputs))
	for _, output := range s.outputs {
		referencePaths = append(referencePaths, filepath.Join(s.OutputDir, output.Filename))
	}
	s.mu.RUnlock()
	for _, referencePath := range referencePaths {
		if _, err := os.Stat(referencePath); err != nil {
			continue
		}
		fingerprint, err := s.FFmpegService.FingerprintVideo(referencePath)
		if err != nil {
			s.updateTaskStatus(taskID, "failed", 30, fmt.Sprintf("Không thể kiểm tra video đã tạo trước đó %q: %v", filepath.Base(referencePath), err))
			return
		}
		references = append(references, fingerprint)
	}

	for i := 0; i < totalOutputs; i++ {
		progressPct := 20 + int(float64(i)/float64(totalOutputs)*70.0)
		var narrationPath string
		script := ""
		if task.Request.ReplaceVoice {
			s.updateTaskStatus(taskID, "processing", progressPct, fmt.Sprintf("Đang viết kịch bản riêng %d/%d và tạo giọng đọc...", i+1, totalOutputs))
			var err error
			script, err = s.TTSService.GenerateDistinctScript(
				task.Request.ProductDescription,
				task.Request.ScriptStyle,
				montageDuration,
				task.Request.SpeechRate,
				task.Request.Voice,
				i+1,
				scripts,
			)
			if err != nil {
				s.updateTaskStatus(taskID, "failed", progressPct, fmt.Sprintf("Không thể viết kịch bản %d bằng Groq: %v", i+1, err))
				return
			}

			if isDuplicateScript(script, scripts) {
				s.updateTaskStatus(taskID, "failed", progressPct, fmt.Sprintf("Groq đã trả về kịch bản trùng cho video %d; tiến trình dừng để không tạo video có lời thoại lặp.", i+1))
				return
			}
			scripts = append(scripts, script)
			s.mu.Lock()
			if currentTask, ok := s.tasks[taskID]; ok {
				currentTask.Scripts = append([]string(nil), scripts...)
				if currentTask.Script == "" {
					currentTask.Script = script
				}
			}
			s.mu.Unlock()
			narrationPath, err = s.TTSService.GenerateSpeechForDuration(script, task.Request.Voice, task.Request.SpeechRate, montageDuration)
			if err != nil {
				s.updateTaskStatus(taskID, "failed", progressPct, fmt.Sprintf("Không thể tạo giọng đọc cho kịch bản %d: %v", i+1, err))
				return
			}
		}

		subtitleSettings := SubtitleSettings{}
		if task.Request.FollowSubtitles && task.Request.ReplaceVoice {
			subtitleSettings = SubtitleSettings{
				Text:     script,
				Position: task.Request.SubtitlePosition,
				Style:    task.Request.SubtitleStyle,
			}
		}
		var bestCandidatePath string
		var bestFingerprint VideoFingerprint
		bestSimilarity := math.MaxFloat64
		for attempt := 1; attempt <= maxNoveltyAttempts; attempt++ {
			msg := fmt.Sprintf("Đang tạo và so sánh phiên bản %d/%d của video %d/%d...", attempt, maxNoveltyAttempts, i+1, totalOutputs)
			s.updateTaskStatus(taskID, "processing", progressPct, msg)
			_, candidatePath, _, err := s.FFmpegService.GenerateRemixVideo(
				i,
				taskID,
				inputPaths,
				task.Request.Duration,
				task.Request.CutSensitivity,
				task.Request.Deduplication,
				narrationPath,
				task.Request.RemixMode,
				subtitleSettings,
				attempt,
			)
			if err != nil {
				if candidatePath != "" {
					_ = os.Remove(candidatePath)
				}
				if bestCandidatePath != "" {
					_ = os.Remove(bestCandidatePath)
				}
				s.updateTaskStatus(taskID, "failed", progressPct, fmt.Sprintf("Lỗi tạo video %d: %v", i+1, err))
				return
			}
			candidateFingerprint, err := s.FFmpegService.FingerprintVideo(candidatePath)
			if err != nil {
				_ = os.Remove(candidatePath)
				if bestCandidatePath != "" {
					_ = os.Remove(bestCandidatePath)
				}
				s.updateTaskStatus(taskID, "failed", progressPct, fmt.Sprintf("Không thể so sánh độ trùng video %d: %v", i+1, err))
				return
			}
			similarity := maxSimilarity(candidateFingerprint, references)
			if similarity < bestSimilarity {
				if bestCandidatePath != "" {
					_ = os.Remove(bestCandidatePath)
				}
				bestCandidatePath = candidatePath
				bestFingerprint = candidateFingerprint
				bestSimilarity = similarity
			} else {
				_ = os.Remove(candidatePath)
			}
			if bestSimilarity < similarityThreshold {
				break
			}
		}
		if bestCandidatePath == "" {
			s.updateTaskStatus(taskID, "failed", progressPct, fmt.Sprintf("Không tạo được video %d.", i+1))
			return
		}
		outFilename := fmt.Sprintf("remix_%s_%d.mp4", taskID[:8], i+1)
		outputPath := filepath.Join(s.OutputDir, outFilename)
		if err := os.Rename(bestCandidatePath, outputPath); err != nil {
			_ = os.Remove(bestCandidatePath)
			s.updateTaskStatus(taskID, "failed", progressPct, fmt.Sprintf("Không thể lưu video khác biệt nhất cho video %d: %v", i+1, err))
			return
		}
		info, err := os.Stat(outputPath)
		if err != nil {
			s.updateTaskStatus(taskID, "failed", progressPct, fmt.Sprintf("Không thể đọc video đầu ra %d: %v", i+1, err))
			return
		}
		if bestSimilarity >= similarityThreshold {
			similarFallbackCount++
		}
		references = append(references, bestFingerprint)
		size := info.Size()
		actualDuration, err := probeDuration(outputPath)
		if err != nil {
			s.updateTaskStatus(taskID, "failed", progressPct, fmt.Sprintf("Không thể đọc thời lượng video đầu ra: %v", err))
			return
		}
		actualDurationSeconds := int(math.Round(actualDuration))

		outVid := model.OutputVideo{
			ID:        generateID(),
			TaskID:    taskID,
			Title:     fmt.Sprintf("Remix #%d (%ds)", i+1, actualDurationSeconds),
			Filename:  outFilename,
			URL:       "/storage/outputs/" + outFilename,
			Duration:  actualDurationSeconds,
			Size:      size,
			Script:    script,
			CreatedAt: time.Now(),
		}

		generatedVideos = append(generatedVideos, outVid)

		s.mu.Lock()
		s.outputs[outVid.ID] = outVid
		persistErr := s.persistOutputs()
		s.mu.Unlock()
		if persistErr != nil {
			s.updateTaskStatus(taskID, "failed", progressPct, fmt.Sprintf("Video đã tạo nhưng không thể lưu danh sách: %v", persistErr))
			return
		}
	}

	s.mu.Lock()
	task.Status = "completed"
	task.Progress = 100
	task.Message = fmt.Sprintf("Hoàn thành! Đã tạo thành công %d video remix.", len(generatedVideos))
	if similarFallbackCount > 0 {
		task.Message += fmt.Sprintf(" Cảnh báo: %d video vẫn có thể giống một video đã tạo trước đó sau %d lần thử; video khác biệt nhất đã được chọn.", similarFallbackCount, maxNoveltyAttempts)
	}
	task.OutputVideos = generatedVideos
	task.UpdatedAt = time.Now()
	s.mu.Unlock()
}

func (s *RemixService) updateTaskStatus(taskID, status string, progress int, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tasks[taskID]; ok {
		t.Status = status
		t.Progress = progress
		t.Message = msg
		t.UpdatedAt = time.Now()
	}
}

func isDuplicateScript(script string, previousScripts []string) bool {
	normalized := normalizeScript(script)
	if normalized == "" {
		return true
	}
	for _, previous := range previousScripts {
		if normalized == normalizeScript(previous) {
			return true
		}
	}
	return false
}

func normalizeScript(script string) string {
	var normalized strings.Builder
	needsSpace := false
	for _, character := range strings.ToLower(script) {
		if unicode.IsLetter(character) || unicode.IsNumber(character) {
			if needsSpace && normalized.Len() > 0 {
				normalized.WriteByte(' ')
			}
			normalized.WriteRune(character)
			needsSpace = false
		} else {
			needsSpace = true
		}
	}
	return normalized.String()
}

// GetTask retrieves a task by ID.
func (s *RemixService) GetTask(taskID string) (*model.RemixTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return nil, false
	}
	taskCopy := *t
	taskCopy.Scripts = append([]string(nil), t.Scripts...)
	taskCopy.OutputVideos = append([]model.OutputVideo(nil), t.OutputVideos...)
	return &taskCopy, true
}

// GetTasks returns all tasks.
func (s *RemixService) GetTasks() []model.RemixTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]model.RemixTask, 0, len(s.tasks))
	for _, t := range s.tasks {
		taskCopy := *t
		taskCopy.Scripts = append([]string(nil), t.Scripts...)
		taskCopy.OutputVideos = append([]model.OutputVideo(nil), t.OutputVideos...)
		result = append(result, taskCopy)
	}
	return result
}

// GetOutputs returns all output videos.
func (s *RemixService) GetOutputs() []model.OutputVideo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]model.OutputVideo, 0, len(s.outputs))
	for _, o := range s.outputs {
		result = append(result, o)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result
}

func (s *RemixService) WriteTaskOutputsZip(taskID string, destination io.Writer) error {
	s.mu.RLock()
	outputs := make([]model.OutputVideo, 0)
	for _, output := range s.outputs {
		if output.TaskID == taskID {
			outputs = append(outputs, output)
		}
	}
	s.mu.RUnlock()
	if len(outputs) == 0 {
		return fmt.Errorf("%w %q", ErrTaskOutputsNotFound, taskID)
	}
	sort.Slice(outputs, func(i, j int) bool {
		return outputs[i].CreatedAt.Before(outputs[j].CreatedAt)
	})

	archive := zip.NewWriter(destination)
	for _, output := range outputs {
		filename := filepath.Base(output.Filename)
		input, err := os.Open(filepath.Join(s.OutputDir, filename))
		if err != nil {
			_ = archive.Close()
			return fmt.Errorf("không thể mở video %q để nén: %w", filename, err)
		}
		entry, err := archive.Create(filename)
		if err != nil {
			_ = input.Close()
			_ = archive.Close()
			return fmt.Errorf("không thể thêm video %q vào tệp nén: %w", filename, err)
		}
		_, copyErr := io.Copy(entry, input)
		closeErr := input.Close()
		if copyErr != nil {
			_ = archive.Close()
			return fmt.Errorf("không thể nén video %q: %w", filename, copyErr)
		}
		if closeErr != nil {
			_ = archive.Close()
			return fmt.Errorf("không thể đóng video %q: %w", filename, closeErr)
		}
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("không thể hoàn tất tệp nén video: %w", err)
	}
	return nil
}
