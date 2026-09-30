package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"ai-ads-studio/backend/internal/model"
)

type TTSService struct {
	AudioDir   string
	ModelDir   string
	Executable string
	GroqAPIKey string
	GroqModel  string
}

func NewTTSService(audioDir string) *TTSService {
	_ = os.MkdirAll(audioDir, 0755)
	modelDir := os.Getenv("PIPER_MODEL_DIR")
	if modelDir == "" {
		modelDir = "models"
		if _, err := os.Stat(modelDir); err != nil {
			if _, rootErr := os.Stat(filepath.Join("backend", modelDir)); rootErr == nil {
				modelDir = filepath.Join("backend", modelDir)
			}
		}
	}
	executable := os.Getenv("PIPER_EXECUTABLE")
	if executable == "" {
		executable = findPiperExecutable()
	}
	groqModel := os.Getenv("GROQ_MODEL")
	if groqModel == "" {
		groqModel = "qwen/qwen3.8-27b"
	}
	return &TTSService{
		AudioDir:   audioDir,
		ModelDir:   modelDir,
		Executable: executable,
		GroqAPIKey: os.Getenv("GROQ_API_KEY"),
		GroqModel:  groqModel,
	}
}

func findPiperExecutable() string {
	if executable, err := exec.LookPath("piper"); err == nil {
		return executable
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		matches, _ := filepath.Glob(filepath.Join(localAppData, "Programs", "Python", "Python*", "Scripts", "piper.exe"))
		if len(matches) > 0 {
			return matches[len(matches)-1]
		}
	}
	return "piper"
}

func (s *TTSService) GetVoices() []model.VoiceOption {
	modelPaths, _ := filepath.Glob(filepath.Join(s.ModelDir, "vi_VN-*.onnx"))
	sort.Strings(modelPaths)
	voices := make([]model.VoiceOption, 0)
	for _, modelPath := range modelPaths {
		modelID := strings.TrimSuffix(filepath.Base(modelPath), filepath.Ext(modelPath))
		configData, err := os.ReadFile(modelPath + ".json")
		if err != nil {
			continue
		}
		var config struct {
			NumSpeakers int            `json:"num_speakers"`
			SpeakerMap  map[string]int `json:"speaker_id_map"`
		}
		if json.Unmarshal(configData, &config) != nil {
			continue
		}
		if config.NumSpeakers > 1 {
			speakerIDs := make([]int, 0, len(config.SpeakerMap))
			speakerNames := make(map[int]string, len(config.SpeakerMap))
			for name, id := range config.SpeakerMap {
				speakerIDs = append(speakerIDs, id)
				speakerNames[id] = name
			}
			sort.Ints(speakerIDs)
			for _, id := range speakerIDs {
				voices = append(voices, model.VoiceOption{
					ID:          fmt.Sprintf("%s#%d", modelID, id),
					Name:        fmt.Sprintf("VIVOS · %s", speakerNames[id]),
					Description: fmt.Sprintf("Giọng tiếng Việt VIVOS, speaker %d", id),
					Gender:      "Không xác định",
					Style:       "VIVOS",
				})
			}
			continue
		}
		voices = append(voices, model.VoiceOption{
			ID:          modelID,
			Name:        modelVoiceName(modelID),
			Description: "Giọng tiếng Việt từ model Piper local",
			Gender:      "Không xác định",
			Style:       modelID,
		})
	}
	return voices
}

func modelVoiceName(modelID string) string {
	switch modelID {
	case "vi_VN-25hours_single-low":
		return "25hours · Tiếng Việt"
	case "vi_VN-vais1000-medium":
		return "VAI · Tiếng Việt"
	default:
		return modelID
	}
}

// GeneratePreview creates a real WAV preview using Piper.
func (s *TTSService) GeneratePreview(req model.TTSPreviewRequest) (*model.TTSPreviewResponse, error) {
	if req.Text == "" {
		req.Text = "Sản phẩm đang được giới thiệu"
	}
	if req.Voice == "" {
		voices := s.GetVoices()
		if len(voices) == 0 {
			return nil, fmt.Errorf("chưa cài model Piper tiếng Việt trong %q", s.ModelDir)
		}
		req.Voice = voices[0].ID
	}
	if req.Rate <= 0 {
		req.Rate = 1.0
	}
	script, err := s.GenerateScript(req.Text, req.Style, float64(req.Duration), req.Rate)
	if err != nil {
		return nil, err
	}

	audioPath, err := s.GenerateSpeech(script, req.Voice, req.Rate)
	if err != nil {
		return nil, err
	}
	filename := filepath.Base(audioPath)
	audioURL := "/storage/tts/" + filename
	return &model.TTSPreviewResponse{
		AudioURL: audioURL,
		Text:     script,
		Voice:    req.Voice,
		Duration: float64(len([]rune(req.Text))) * 0.075 / req.Rate,
	}, nil
}

func (s *TTSService) GenerateSpeech(text, voice string, rate float64) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("speech text is empty")
	}
	if rate <= 0 {
		rate = 1
	}
	if rate < 0.7 {
		rate = 0.7
	} else if rate > 1.3 {
		rate = 1.3
	}
	if voice == "" {
		available := s.GetVoices()
		if len(available) == 0 {
			return "", fmt.Errorf("chưa cài model Piper tiếng Việt trong %q", s.ModelDir)
		}
		voice = available[0].ID
	}
	modelID, speakerText, hasSpeaker := strings.Cut(voice, "#")
	if filepath.Base(modelID) != modelID || !strings.HasPrefix(modelID, "vi_VN-") {
		return "", fmt.Errorf("voice không hợp lệ: %q", voice)
	}
	modelPath := filepath.Join(s.ModelDir, modelID+".onnx")
	if _, err := os.Stat(modelPath); err != nil {
		return "", fmt.Errorf("không tìm thấy model Piper %q", modelID)
	}
	args := []string{"--model", modelPath}
	if hasSpeaker {
		speakerID, err := strconv.Atoi(speakerText)
		if err != nil || speakerID < 0 {
			return "", fmt.Errorf("speaker ID không hợp lệ trong voice %q", voice)
		}
		args = append(args, "--speaker", strconv.Itoa(speakerID))
	}

	hash := md5.Sum([]byte(fmt.Sprintf("%s_%s_%.2f", text, voice, rate)))
	audioPath := filepath.Join(s.AudioDir, fmt.Sprintf("speech_%s.wav", hex.EncodeToString(hash[:8])))
	if info, err := os.Stat(audioPath); err == nil && info.Size() > 44 {
		return audioPath, nil
	}
	inputFile, err := os.CreateTemp(s.AudioDir, "speech-input-*.txt")
	if err != nil {
		return "", err
	}
	inputPath := inputFile.Name()
	defer os.Remove(inputPath)
	if _, err := io.WriteString(inputFile, text); err != nil {
		inputFile.Close()
		return "", err
	}
	if err := inputFile.Close(); err != nil {
		return "", err
	}
	args = append(args, "--input_file", inputPath, "--output_file", audioPath, "--length_scale", strconv.FormatFloat(1/rate, 'f', 3, 64))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.Executable, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(audioPath)
		return "", fmt.Errorf("Piper không tạo được giọng đọc: %w: %s", err, strings.TrimSpace(string(output)))
	}
	info, err := os.Stat(audioPath)
	if err != nil || info.Size() <= 44 {
		_ = os.Remove(audioPath)
		return "", fmt.Errorf("Piper không tạo được WAV hợp lệ: %s", strings.TrimSpace(string(output)))
	}
	return audioPath, nil
}

// GenerateScript uses Groq to write a short Vietnamese advertisement script.
func (s *TTSService) GenerateScript(productDescription, style string, duration float64, speechRate float64) (string, error) {
	if strings.TrimSpace(s.GroqAPIKey) == "" {
		return "", fmt.Errorf("chưa cấu hình GROQ_API_KEY; hãy thêm khóa vào backend/.env rồi khởi động lại backend")
	}
	if duration <= 0 {
		duration = 30
	}
	if speechRate < 0.7 || speechRate > 1.3 {
		speechRate = 1
	}
	targetWords := int(math.Round(duration * 2.4 * speechRate))
	if targetWords < 15 {
		targetWords = 15
	}
	if targetWords > 200 {
		targetWords = 200
	}

	styleDescription := map[string]string{
		"professional": "chuyên nghiệp, rõ ràng và đáng tin cậy",
		"friendly":     "thân thiện, gần gũi như đang trò chuyện",
		"energetic":    "năng lượng, nhịp nhanh và hào hứng",
		"storytelling": "kể chuyện tự nhiên, giàu cảm xúc nhưng ngắn gọn",
	}[style]
	if styleDescription == "" {
		styleDescription = "chuyên nghiệp, rõ ràng và đáng tin cậy"
	}
	if strings.TrimSpace(productDescription) == "" {
		productDescription = "một sản phẩm chất lượng dành cho khách hàng"
	}

	messages := []map[string]string{
		{
			"role":    "system",
			"content": "Bạn là copywriter quảng cáo tiếng Việt. Viết lời thoại tự nhiên phù hợp chính xác với thời lượng và mục tiêu số tiếng trong yêu cầu. Đếm mỗi cụm phân tách bằng dấu cách là một tiếng; giữ độ dài trong khoảng 90%-110% mục tiêu. Chỉ nêu dữ kiện được ghi rõ trong mô tả sản phẩm. Không suy diễn công dụng, chất liệu, chất lượng, kết quả, giá, khuyến mãi, chứng nhận, bảo hành hoặc cam kết. Nếu thiếu thông tin, giới thiệu trung tính. Không dùng Markdown, tiêu đề hay giải thích; chỉ trả về lời thoại.",
		},
		{
			"role":    "user",
			"content": fmt.Sprintf("Phong cách: %s\nThời lượng video: %.1f giây\nTốc độ đọc: %.1fx\nĐộ dài mục tiêu: khoảng %d tiếng tiếng Việt được phân tách bằng dấu cách.\nMô tả sản phẩm: %s", styleDescription, duration, speechRate, targetWords, strings.TrimSpace(productDescription)),
		},
	}
	requestBody := map[string]any{"model": s.GroqModel, "messages": messages}
	maxTokens := min(2500, max(500, targetWords*8+300))
	if strings.HasPrefix(strings.ToLower(s.GroqModel), "openai/gpt-oss-") {
		requestBody["reasoning_effort"] = "low"
	} else {
		requestBody["temperature"] = 0.3
	}

	var result struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	client := &http.Client{Timeout: 45 * time.Second}
	for attempt := 0; attempt < 2; attempt++ {
		if strings.HasPrefix(strings.ToLower(s.GroqModel), "openai/gpt-oss-") {
			requestBody["max_completion_tokens"] = maxTokens
		} else {
			requestBody["max_tokens"] = maxTokens
		}
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return "", fmt.Errorf("không thể tạo yêu cầu Groq: %w", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(payload))
		if err != nil {
			cancel()
			return "", fmt.Errorf("không thể tạo yêu cầu Groq: %w", err)
		}
		request.Header.Set("Authorization", "Bearer "+s.GroqAPIKey)
		request.Header.Set("Content-Type", "application/json")
		response, err := client.Do(request)
		if err != nil {
			cancel()
			return "", fmt.Errorf("không kết nối được Groq: %w", err)
		}
		responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		response.Body.Close()
		cancel()
		if readErr != nil {
			return "", fmt.Errorf("không đọc được phản hồi Groq: %w", readErr)
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return "", fmt.Errorf("Groq trả HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
		}
		if err := json.Unmarshal(responseBody, &result); err != nil {
			return "", fmt.Errorf("phản hồi Groq không hợp lệ: %w", err)
		}
		if len(result.Choices) == 0 {
			return "", fmt.Errorf("Groq không trả về choice nào")
		}

		choice := result.Choices[0]
		script := strings.TrimSpace(choice.Message.Content)
		if script == "" {
			return "", fmt.Errorf("Groq không trả về kịch bản (finish_reason=%s)", choice.FinishReason)
		}
		if choice.FinishReason != "length" && hasCompleteSentenceEnding(script) {
			return script, nil
		}
		if attempt == 1 {
			return "", fmt.Errorf("Groq vẫn trả kịch bản chưa trọn câu sau lần thử lại (finish_reason=%s)", choice.FinishReason)
		}

		messages = append(messages,
			map[string]string{"role": "assistant", "content": script},
			map[string]string{"role": "user", "content": fmt.Sprintf("Hãy viết lại toàn bộ kịch bản thành lời thoại hoàn chỉnh trong khoảng %d tiếng. Kết thúc bằng dấu chấm, chấm hỏi hoặc chấm than. Không dừng giữa câu; chỉ trả về bản hoàn chỉnh.", targetWords)},
		)
		requestBody["messages"] = messages
		maxTokens = min(4000, maxTokens*2)
	}
	return "", fmt.Errorf("không thể tạo kịch bản hoàn chỉnh")
}

func hasCompleteSentenceEnding(script string) bool {
	trimmed := strings.TrimRight(strings.TrimSpace(script), "\"'”’»)]}")
	if trimmed == "" {
		return false
	}
	last := []rune(trimmed)[len([]rune(trimmed))-1]
	return strings.ContainsRune(".!?…。！？", last)
}
