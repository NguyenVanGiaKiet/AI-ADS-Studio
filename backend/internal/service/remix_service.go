package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ai-ads-studio/backend/internal/model"
)

type RemixService struct {
	UploadDir     string
	OutputDir     string
	FFmpegService *FFmpegService
	TTSService    *TTSService

	mu       sync.RWMutex
	uploads  map[string]model.UploadedVideo
	tasks    map[string]*model.RemixTask
	outputs  map[string]model.OutputVideo
}

func NewRemixService(uploadDir, outputDir string, ffmpegSvc *FFmpegService, ttsSvc *TTSService) *RemixService {
	os.MkdirAll(uploadDir, 0755)
	os.MkdirAll(outputDir, 0755)

	return &RemixService{
		UploadDir:     uploadDir,
		OutputDir:     outputDir,
		FFmpegService: ffmpegSvc,
		TTSService:    ttsSvc,
		uploads:       make(map[string]model.UploadedVideo),
		tasks:         make(map[string]*model.RemixTask),
		outputs:       make(map[string]model.OutputVideo),
	}
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
	if req.AspectRatio == "" {
		req.AspectRatio = "vertical"
	}
	if req.RemixMode == "" {
		req.RemixMode = "standard"
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

	// Update progress: AI script generation if voice replacement active
	if task.Request.ReplaceVoice && task.Request.ProductDescription != "" {
		s.updateTaskStatus(taskID, "processing", 25, "AI đang tổng hợp kịch bản & giọng đọc...")
		_ = s.TTSService.GenerateScript(task.Request.ProductDescription, task.Request.ScriptStyle)
		time.Sleep(600 * time.Millisecond)
	}

	totalOutputs := task.Request.OutputCount
	var generatedVideos []model.OutputVideo

	for i := 0; i < totalOutputs; i++ {
		progressPct := 30 + int(float64(i+1)/float64(totalOutputs)*60.0)
		msg := fmt.Sprintf("Đang cắt ghép & chống trùng video %d/%d...", i+1, totalOutputs)
		s.updateTaskStatus(taskID, "processing", progressPct, msg)

		outFilename, _, size, err := s.FFmpegService.GenerateRemixVideo(
			i,
			taskID,
			inputPaths,
			task.Request.Duration,
			task.Request.AspectRatio,
			task.Request.Deduplication,
		)

		if err != nil {
			s.updateTaskStatus(taskID, "failed", progressPct, fmt.Sprintf("Lỗi tạo video %d: %v", i+1, err))
			return
		}

		outVid := model.OutputVideo{
			ID:        generateID(),
			TaskID:    taskID,
			Title:     fmt.Sprintf("Remix #%d (%s - %ds)", i+1, task.Request.AspectRatio, task.Request.Duration),
			Filename:  outFilename,
			URL:       "/storage/outputs/" + outFilename,
			Duration:  task.Request.Duration,
			Size:      size,
			CreatedAt: time.Now(),
		}

		generatedVideos = append(generatedVideos, outVid)

		s.mu.Lock()
		s.outputs[outVid.ID] = outVid
		s.mu.Unlock()
	}

	s.mu.Lock()
	task.Status = "completed"
	task.Progress = 100
	task.Message = fmt.Sprintf("Hoàn thành! Đã tạo thành công %d video remix.", len(generatedVideos))
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

// GetTask retrieves a task by ID.
func (s *RemixService) GetTask(taskID string) (*model.RemixTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[taskID]
	return t, ok
}

// GetTasks returns all tasks.
func (s *RemixService) GetTasks() []model.RemixTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]model.RemixTask, 0, len(s.tasks))
	for _, t := range s.tasks {
		result = append(result, *t)
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
	return result
}
