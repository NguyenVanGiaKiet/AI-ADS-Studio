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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-ads-studio/backend/internal/model"
)

type TTSService struct {
	AudioDir        string
	GroqAPIKey      string
	GroqModel       string
	ElevenLabsKey   string
	ElevenLabsModel string
}

func NewTTSService(audioDir string) *TTSService {
	_ = os.MkdirAll(audioDir, 0755)
	groqModel := os.Getenv("GROQ_MODEL")
	if groqModel == "" {
		groqModel = "qwen/qwen3.8-27b"
	}
	elevenLabsModel := os.Getenv("ELEVENLABS_MODEL_ID")
	if elevenLabsModel == "" {
		elevenLabsModel = "eleven_multilingual_v2"
	}
	return &TTSService{
		AudioDir:        audioDir,
		GroqAPIKey:      os.Getenv("GROQ_API_KEY"),
		GroqModel:       groqModel,
		ElevenLabsKey:   os.Getenv("ELEVENLABS_API_KEY"),
		ElevenLabsModel: elevenLabsModel,
	}
}

func (s *TTSService) GetVoices() ([]model.VoiceOption, error) {
	if strings.TrimSpace(s.ElevenLabsKey) == "" {
		return nil, fmt.Errorf("chưa cấu hình ELEVENLABS_API_KEY trong backend/.env")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	var voices []model.VoiceOption
	nextPageToken := ""
	for page := 0; page < 10; page++ {
		query := url.Values{"page_size": {"100"}}
		if nextPageToken != "" {
			query.Set("next_page_token", nextPageToken)
		}
		request, err := http.NewRequest(http.MethodGet, "https://api.elevenlabs.io/v2/voices?"+query.Encode(), nil)
		if err != nil {
			return nil, fmt.Errorf("không thể tạo yêu cầu lấy voice ElevenLabs: %w", err)
		}
		request.Header.Set("xi-api-key", s.ElevenLabsKey)
		response, err := client.Do(request)
		if err != nil {
			return nil, fmt.Errorf("không kết nối được ElevenLabs: %w", err)
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
		response.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("không đọc được danh sách voice ElevenLabs: %w", readErr)
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return nil, fmt.Errorf("ElevenLabs trả HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
		}
		var result struct {
			Voices []struct {
				ID          string            `json:"voice_id"`
				Name        string            `json:"name"`
				Description string            `json:"description"`
				Labels      map[string]string `json:"labels"`
			} `json:"voices"`
			NextPageToken string `json:"next_page_token"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("response danh sách voice ElevenLabs không hợp lệ: %w", err)
		}
		for _, voice := range result.Voices {
			if voice.ID == "" || voice.Name == "" {
				continue
			}
			voices = append(voices, model.VoiceOption{
				ID:          voice.ID,
				Name:        voice.Name,
				Description: voice.Description,
				Gender:      voice.Labels["gender"],
				Style:       voice.Labels["accent"],
			})
		}
		nextPageToken = result.NextPageToken
		if nextPageToken == "" {
			break
		}
	}
	return voices, nil
}

// GeneratePreview creates speech and returns the audio URL and generated script.
func (s *TTSService) GeneratePreview(req model.TTSPreviewRequest) (*model.TTSPreviewResponse, error) {
	if req.Text == "" {
		req.Text = "Sản phẩm đang được giới thiệu"
	}
	if req.Voice == "" {
		voices, err := s.GetVoices()
		if err != nil {
			return nil, err
		}
		if len(voices) == 0 {
			return nil, fmt.Errorf("tài khoản ElevenLabs chưa có voice nào")
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
		Duration: float64(len([]rune(script))) * 0.075 / req.Rate,
	}, nil
}

func (s *TTSService) GenerateSpeech(text, voice string, rate float64) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("speech text is empty")
	}
	if strings.TrimSpace(s.ElevenLabsKey) == "" {
		return "", fmt.Errorf("chưa cấu hình ELEVENLABS_API_KEY trong backend/.env")
	}
	if rate <= 0 {
		rate = 1
	}
	if rate < 0.7 {
		rate = 0.7
	} else if rate > 1.2 {
		rate = 1.2
	}
	if voice == "" {
		available, err := s.GetVoices()
		if err != nil {
			return "", err
		}
		if len(available) == 0 {
			return "", fmt.Errorf("tài khoản ElevenLabs chưa có voice nào")
		}
		voice = available[0].ID
	}
	hash := md5.Sum([]byte(fmt.Sprintf("%s_%s_%s_%.2f", text, voice, s.ElevenLabsModel, rate)))
	audioPath := filepath.Join(s.AudioDir, fmt.Sprintf("speech_%s.mp3", hex.EncodeToString(hash[:8])))
	if info, err := os.Stat(audioPath); err == nil && info.Size() > 128 {
		return audioPath, nil
	}
	requestBody := struct {
		Text          string `json:"text"`
		ModelID       string `json:"model_id"`
		VoiceSettings struct {
			Speed float64 `json:"speed"`
		} `json:"voice_settings"`
	}{Text: text, ModelID: s.ElevenLabsModel}
	requestBody.VoiceSettings.Speed = rate
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("không thể tạo yêu cầu ElevenLabs: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	endpoint := "https://api.elevenlabs.io/v1/text-to-speech/" + url.PathEscape(voice) + "?output_format=mp3_44100_128"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("không thể tạo yêu cầu ElevenLabs: %w", err)
	}
	request.Header.Set("xi-api-key", s.ElevenLabsKey)
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 2 * time.Minute}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("không kết nối được ElevenLabs: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		return "", fmt.Errorf("ElevenLabs trả HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	audio, err := io.ReadAll(io.LimitReader(response.Body, 64<<20))
	if err != nil {
		return "", fmt.Errorf("không đọc được audio ElevenLabs: %w", err)
	}
	if len(audio) <= 128 {
		return "", fmt.Errorf("ElevenLabs trả về audio rỗng hoặc không hợp lệ")
	}
	if err := os.WriteFile(audioPath, audio, 0600); err != nil {
		_ = os.Remove(audioPath)
		return "", fmt.Errorf("không thể lưu audio ElevenLabs: %w", err)
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
