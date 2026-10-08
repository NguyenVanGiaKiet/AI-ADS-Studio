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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"ai-ads-studio/backend/internal/model"
)

const (
	freeVietnameseVoiceID  = "free-vi-vn"
	vietnameseVoiceModel   = "vi_VN-vais1000-medium.onnx"
	vietnameseVivosModel   = "vi_VN-vivos-x_low.onnx"
	vietnamese25HoursModel = "vi_VN-25hours_single-low.onnx"
	voicePreviewText       = "AI ADS Studio là nền tảng AI giúp tự động hóa quy trình tạo video quảng cáo chuyên nghiệp từ hình ảnh và thông tin sản phẩm, nhanh chóng, dễ dàng và tiết kiệm chi phí."
	voiceCalibrationText   = "Hôm nay chúng ta cùng khám phá sản phẩm mới, tiện lợi và phù hợp cho cả gia đình."
)

type TTSService struct {
	AudioDir           string
	GroqAPIKey         string
	GroqModel          string
	PiperPython        string
	VoiceModelPath     string
	calibrationMu      sync.Mutex
	syllablesPerSecond map[string]float64
}

func NewTTSService(audioDir string) *TTSService {
	groqModel := os.Getenv("GROQ_MODEL")
	if groqModel == "" {
		groqModel = "qwen/qwen3.8-27b"
	}
	piperPython := strings.TrimSpace(os.Getenv("PIPER_PYTHON"))
	if piperPython == "" {
		piperPython = "python"
	}
	voiceModelPath := strings.TrimSpace(os.Getenv("PIPER_MODEL_PATH"))
	if voiceModelPath == "" {
		voiceModelPath = findBundledVoiceModel()
	}
	return &TTSService{
		AudioDir:       audioDir,
		GroqAPIKey:     os.Getenv("GROQ_API_KEY"),
		GroqModel:      groqModel,
		PiperPython:    piperPython,
		VoiceModelPath: voiceModelPath,
	}
}

func findBundledVoiceModel() string {
	candidates := []string{
		filepath.Join("models", vietnameseVoiceModel),
		filepath.Join("backend", "models", vietnameseVoiceModel),
	}
	if _, sourcePath, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(sourcePath), "..", "..", "models", vietnameseVoiceModel))
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			if absolutePath, err := filepath.Abs(candidate); err == nil {
				return absolutePath
			}
			return candidate
		}
	}
	return candidates[0]
}

func freeVietnameseVoice() model.VoiceOption {
	return model.VoiceOption{
		ID:          freeVietnameseVoiceID,
		Name:        "Tiếng Việt — VAI 1000 (Piper, chạy cục bộ)",
		Description: "Giọng tiếng Việt chạy trực tiếp trong backend bằng model Piper có sẵn.",
		Gender:      "neutral",
		Style:       "Vietnamese",
	}
}

func (s *TTSService) GetVoices() ([]model.VoiceOption, error) {
	voices := make([]model.VoiceOption, 0, 67)
	primaryModelPath := s.VoiceModelPath
	if primaryModelPath == "" {
		primaryModelPath = findBundledVoiceModel()
	}
	if _, err := os.Stat(primaryModelPath); err == nil {
		voices = append(voices, freeVietnameseVoice())
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("không thể kiểm tra model giọng Piper mặc định %q: %w", primaryModelPath, err)
	}

	modelsDir := filepath.Dir(findBundledVoiceModel())
	hoursModelPath := filepath.Join(modelsDir, vietnamese25HoursModel)
	if _, err := os.Stat(hoursModelPath); err == nil {
		voices = append(voices, model.VoiceOption{
			ID:          "vi-vn-25hours-single",
			Name:        "Tiếng Việt — 25Hours (giọng đơn)",
			Description: "Giọng tiếng Việt Piper 25Hours, chạy cục bộ.",
			Gender:      "neutral",
			Style:       "Vietnamese",
		})
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("không thể kiểm tra model Piper 25Hours %q: %w", hoursModelPath, err)
	}

	vivosModelPath := filepath.Join(modelsDir, vietnameseVivosModel)
	if _, err := os.Stat(vivosModelPath); err == nil {
		speakers, err := loadPiperSpeakers(vivosModelPath)
		if err != nil {
			return nil, err
		}
		for index, speaker := range speakers {
			voices = append(voices, model.VoiceOption{
				ID:          fmt.Sprintf("vi-vn-vivos-%d", index),
				Name:        fmt.Sprintf("VIVOS — Giọng %02d", index+1),
				Description: fmt.Sprintf("Giọng %s trong model VIVOS tiếng Việt chạy cục bộ.", speaker),
				Gender:      "neutral",
				Style:       "Vietnamese",
			})
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("không thể kiểm tra model Piper VIVOS %q: %w", vivosModelPath, err)
	}

	if len(voices) == 0 {
		return nil, fmt.Errorf("không tìm thấy model Piper tiếng Việt; hãy kiểm tra thư mục backend/models")
	}
	return voices, nil
}

func loadPiperSpeakers(modelPath string) ([]string, error) {
	configPath := modelPath + ".json"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("không đọc được cấu hình giọng VIVOS %q: %w", configPath, err)
	}
	var config struct {
		SpeakerIDMap map[string]int `json:"speaker_id_map"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("cấu hình giọng VIVOS không hợp lệ: %w", err)
	}
	if len(config.SpeakerIDMap) == 0 {
		return nil, fmt.Errorf("model VIVOS không khai báo speaker_id_map")
	}
	speakers := make([]string, len(config.SpeakerIDMap))
	for speaker, id := range config.SpeakerIDMap {
		if id < 0 || id >= len(speakers) || speakers[id] != "" {
			return nil, fmt.Errorf("speaker_id_map của VIVOS không hợp lệ")
		}
		speakers[id] = speaker
	}
	return speakers, nil
}

func (s *TTSService) resolveVoice(voiceID string) (string, *int, error) {
	modelsDir := filepath.Dir(findBundledVoiceModel())
	switch {
	case voiceID == "" || voiceID == freeVietnameseVoiceID:
		modelPath := s.VoiceModelPath
		if modelPath == "" {
			modelPath = findBundledVoiceModel()
		}
		return modelPath, nil, nil
	case voiceID == "vi-vn-25hours-single":
		return filepath.Join(modelsDir, vietnamese25HoursModel), nil, nil
	case strings.HasPrefix(voiceID, "vi-vn-vivos-"):
		speakerID, err := strconv.Atoi(strings.TrimPrefix(voiceID, "vi-vn-vivos-"))
		if err != nil {
			return "", nil, fmt.Errorf("mã giọng VIVOS không hợp lệ: %q", voiceID)
		}
		speakers, err := loadPiperSpeakers(filepath.Join(modelsDir, vietnameseVivosModel))
		if err != nil {
			return "", nil, err
		}
		if speakerID < 0 || speakerID >= len(speakers) {
			return "", nil, fmt.Errorf("không tìm thấy giọng VIVOS có mã %d", speakerID)
		}
		return filepath.Join(modelsDir, vietnameseVivosModel), &speakerID, nil
	default:
		return "", nil, fmt.Errorf("giọng đọc %q không được hỗ trợ; hãy chọn một giọng Piper tiếng Việt", voiceID)
	}
}

// GeneratePreview creates a local Vietnamese speech preview and its script.
func (s *TTSService) GeneratePreview(req model.TTSPreviewRequest) (*model.TTSPreviewResponse, error) {
	if req.Voice == "" {
		req.Voice = freeVietnameseVoiceID
	}
	if req.Rate <= 0 {
		req.Rate = 1
	}
	audioPath, err := s.GenerateSpeechForDuration(voicePreviewText, req.Voice, req.Rate, 0)
	if err != nil {
		return nil, err
	}
	audioDuration, err := probeDuration(audioPath)
	if err != nil {
		return nil, fmt.Errorf("không thể xác định thời lượng audio preview: %w", err)
	}
	return &model.TTSPreviewResponse{
		AudioURL: "/storage/tts/" + filepath.Base(audioPath),
		Text:     voicePreviewText,
		Voice:    req.Voice,
		Duration: audioDuration,
	}, nil
}

func (s *TTSService) GenerateSpeech(text, voice string, rate float64) (string, error) {
	return s.GenerateSpeechForDuration(text, voice, rate, 0)
}

func (s *TTSService) GenerateSpeechForDuration(text, voice string, rate, targetDuration float64) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
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
	if err := os.MkdirAll(s.AudioDir, 0755); err != nil {
		return "", fmt.Errorf("không thể tạo thư mục lưu giọng đọc: %w", err)
	}
	modelPath, speakerID, err := s.resolveVoice(voice)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(modelPath); err != nil {
		return "", fmt.Errorf("không tìm thấy model giọng đọc tiếng Việt cục bộ %q: %w", modelPath, err)
	}

	hash := md5.Sum([]byte(fmt.Sprintf("%s_%s_%v_%.2f_%.3f", text, modelPath, speakerID, rate, targetDuration)))
	audioPath := filepath.Join(s.AudioDir, fmt.Sprintf("speech_%s.wav", hex.EncodeToString(hash[:8])))
	rawAudioPath := filepath.Join(s.AudioDir, fmt.Sprintf("speech_%s_raw.wav", hex.EncodeToString(hash[:8])))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.PiperPython,
		"-m", "piper",
		"-m", modelPath,
		"-f", rawAudioPath,
	)
	if speakerID != nil {
		cmd.Args = append(cmd.Args, "--speaker", strconv.Itoa(*speakerID))
	}
	cmd.Args = append(cmd.Args, "--", text)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(rawAudioPath)
		if ctx.Err() != nil {
			return "", fmt.Errorf("Piper TTS chạy quá thời gian cho phép: %w", ctx.Err())
		}
		return "", fmt.Errorf("không thể chạy Piper TTS cục bộ (hãy cài piper-tts vào Python backend): %w: %s", err, strings.TrimSpace(string(output)))
	}
	info, err := os.Stat(rawAudioPath)
	if err != nil {
		return "", fmt.Errorf("Piper không tạo được tệp giọng đọc: %w", err)
	}
	if info.Size() <= 44 {
		return "", fmt.Errorf("Piper tạo ra tệp giọng đọc rỗng hoặc không hợp lệ")
	}
	rawDuration, err := probeDuration(rawAudioPath)
	if err != nil {
		_ = os.Remove(rawAudioPath)
		return "", fmt.Errorf("không thể đo thời lượng giọng đọc Piper: %w", err)
	}
	tempo := rate
	if targetDuration > 0 {
		tempo = fitTempoFactor(rawDuration, targetDuration, rate)
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		_ = os.Remove(rawAudioPath)
		return "", fmt.Errorf("cần cài FFmpeg để hoàn tất giọng đọc đúng thời lượng: %w", err)
	}
	audioFilters := []string{buildAudioTempoFilter(tempo)}
	if targetDuration > 0 {
		audioFilters = append(audioFilters,
			fmt.Sprintf("apad=whole_dur=%.6f", targetDuration),
			fmt.Sprintf("atrim=duration=%.6f", targetDuration),
		)
	}
	cmd = exec.CommandContext(ctx, ffmpeg,
		"-hide_banner", "-loglevel", "error", "-y",
		"-i", rawAudioPath,
		"-filter:a", strings.Join(audioFilters, ","),
		"-c:a", "pcm_s16le", audioPath,
	)
	output, err = cmd.CombinedOutput()
	_ = os.Remove(rawAudioPath)
	if err != nil {
		_ = os.Remove(audioPath)
		return "", fmt.Errorf("không thể căn thời lượng giọng đọc bằng FFmpeg: %w: %s", err, strings.TrimSpace(string(output)))
	}
	info, err = os.Stat(audioPath)
	if err != nil {
		return "", fmt.Errorf("không tìm thấy tệp giọng đọc sau khi tạo: %w", err)
	}
	if info.Size() <= 44 {
		return "", fmt.Errorf("tệp giọng đọc sau khi tạo rỗng hoặc không hợp lệ")
	}
	return audioPath, nil
}

func fitTempoFactor(sourceDuration, targetDuration, requestedRate float64) float64 {
	if sourceDuration <= 0 {
		if requestedRate > 0 {
			return requestedRate
		}
		return 1
	}
	if targetDuration <= 0 {
		if requestedRate > 0 {
			return requestedRate
		}
		return 1
	}
	return sourceDuration / targetDuration * 1.005
}

func buildAudioTempoFilter(tempo float64) string {
	if tempo <= 0 {
		tempo = 1
	}
	var filters []string
	for tempo > 2 {
		filters = append(filters, "atempo=2.000")
		tempo /= 2
	}
	for tempo < 0.5 {
		filters = append(filters, "atempo=0.500")
		tempo /= 0.5
	}
	filters = append(filters, fmt.Sprintf("atempo=%.6f", tempo))
	return strings.Join(filters, ",")
}

// GenerateScript uses Groq to write a short Vietnamese advertisement script.
func (s *TTSService) GenerateScript(productDescription, style string, duration float64, speechRate float64) (string, error) {
	return s.generateScript(productDescription, style, duration, speechRate, "", 0)
}

func (s *TTSService) GenerateDistinctScript(productDescription, style string, duration float64, speechRate float64, voice string, variant int, previousScripts []string) (string, error) {
	approaches := []string{
		"Mở đầu bằng một tình huống đời thường của khách hàng, rồi dẫn tự nhiên tới sản phẩm.",
		"Mở đầu bằng cách gọi tên nhu cầu hoặc vấn đề của khách hàng, sau đó giới thiệu sản phẩm.",
		"Mở đầu bằng một câu gợi tò mò về trải nghiệm mua sắm, rồi tập trung vào điểm đã nêu trong mô tả.",
		"Mở đầu bằng lời trò chuyện trực tiếp với người xem, kết thúc bằng lời mời tìm hiểu sản phẩm.",
		"Mở đầu bằng một nhận xét ngắn gọn về lựa chọn mua sắm, rồi giới thiệu sản phẩm theo cách mới.",
	}
	if variant < 1 {
		return "", fmt.Errorf("số thứ tự video phải lớn hơn 0")
	}
	syllablesPerSecond, err := s.estimateSyllablesPerSecond(voice)
	if err != nil {
		return "", fmt.Errorf("không thể hiệu chuẩn tốc độ tự nhiên của giọng đọc: %w", err)
	}
	targetWords := scriptTargetWords(duration, speechRate, syllablesPerSecond)
	approach := approaches[(variant-1)%len(approaches)]
	previous := append([]string(nil), previousScripts...)
	for attempt := 1; attempt <= 3; attempt++ {
		variation := fmt.Sprintf(
			"Đây là video số %d trong một bộ nhiều video. Hướng triển khai riêng: %s\n"+
				"Viết mới hoàn toàn, không lặp lại câu mở đầu, cách triển khai hay câu kết của các kịch bản trước. "+
				"Không được sao chép nguyên câu hoặc diễn đạt lại sát nghĩa các kịch bản sau:\n%s",
			variant, approach, strings.Join(previous, "\n---\n"),
		)
		if attempt > 1 {
			variation += fmt.Sprintf("\nLần viết lại %d: hãy đổi cách diễn đạt rõ rệt và bảo đảm kịch bản khác tất cả nội dung liệt kê.", attempt)
		}
		script, err := s.generateScript(productDescription, style, duration, speechRate, variation, targetWords)
		if err != nil {
			return "", err
		}
		if !isDuplicateScript(script, previous) {
			return script, nil
		}
		previous = append(previous, script)
	}
	return "", fmt.Errorf("Groq trả về kịch bản trùng sau 3 lần thử; chưa thể bảo đảm lời thoại riêng cho video %d", variant)
}

func (s *TTSService) estimateSyllablesPerSecond(voice string) (float64, error) {
	modelPath, speakerID, err := s.resolveVoice(voice)
	if err != nil {
		return 0, err
	}
	cacheKey := fmt.Sprintf("%s:%v", modelPath, speakerID)

	s.calibrationMu.Lock()
	defer s.calibrationMu.Unlock()
	if rate, ok := s.syllablesPerSecond[cacheKey]; ok {
		return rate, nil
	}

	audioPath, err := s.GenerateSpeechForDuration(voiceCalibrationText, voice, 1, 0)
	if err != nil {
		return 0, err
	}
	defer os.Remove(audioPath)
	duration, err := probeDuration(audioPath)
	if err != nil {
		return 0, fmt.Errorf("không thể đo audio hiệu chuẩn: %w", err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("audio hiệu chuẩn có thời lượng không hợp lệ")
	}
	measuredRate := float64(len(strings.Fields(voiceCalibrationText))) / duration
	if measuredRate <= 0 || math.IsNaN(measuredRate) || math.IsInf(measuredRate, 0) {
		return 0, fmt.Errorf("tốc độ tiếng/giây đo được không hợp lệ")
	}
	if s.syllablesPerSecond == nil {
		s.syllablesPerSecond = make(map[string]float64)
	}
	s.syllablesPerSecond[cacheKey] = measuredRate
	return measuredRate, nil
}

func scriptTargetWords(duration, speechRate, syllablesPerSecond float64) int {
	if duration <= 0 {
		duration = 30
	}
	if speechRate < 0.7 || speechRate > 1.3 || math.IsNaN(speechRate) {
		speechRate = 1
	}
	if syllablesPerSecond <= 0 || math.IsNaN(syllablesPerSecond) || math.IsInf(syllablesPerSecond, 0) {
		syllablesPerSecond = 2.4
	}
	targetWords := int(math.Round(duration * speechRate * syllablesPerSecond))
	return min(500, max(15, targetWords))
}

func (s *TTSService) generateScript(productDescription, style string, duration float64, speechRate float64, variation string, requestedTargetWords int) (string, error) {
	if strings.TrimSpace(s.GroqAPIKey) == "" {
		return "", fmt.Errorf("chưa cấu hình GROQ_API_KEY trong backend/.env")
	}
	if duration <= 0 {
		duration = 30
	}
	if speechRate < 0.7 || speechRate > 1.3 {
		speechRate = 1
	}
	targetWords := requestedTargetWords
	if targetWords <= 0 {
		targetWords = scriptTargetWords(duration, speechRate, 2.4)
	}
	styleDescription := scriptStyleDescription(style)
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
			"content": fmt.Sprintf("%s\nPhong cách: %s\nThời lượng video: %.1f giây\nTốc độ đọc: %.1fx\nĐộ dài mục tiêu: khoảng %d tiếng tiếng Việt được phân tách bằng dấu cách.\nMô tả sản phẩm: %s", variation, styleDescription, duration, speechRate, targetWords, strings.TrimSpace(productDescription)),
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

func scriptStyleDescription(style string) string {
	description := map[string]string{
		"professional": "chuyên nghiệp, rõ ràng và đáng tin cậy",
		"friendly":     "thân thiện, gần gũi như đang trò chuyện",
		"energetic":    "năng lượng, nhịp nhanh và hào hứng",
		"storytelling": "kể chuyện tự nhiên, giàu cảm xúc nhưng ngắn gọn",
		"adam_drama":   "hài hước, có tình huống drama đời thường và đối thoại duyên dáng; người kể xưng anh hoặc chồng khi phù hợp, không lạm dụng và không biến thành cãi vã",
		"adam_viral":   "phong cách quảng cáo viral vui nhộn, nhịp nhanh, câu ngắn, có cảm thán mạnh và bất ngờ; chỉ dùng cảm thán tự nhiên như 'Trời má', 'Đậu xanh' khi phù hợp, tránh lặp hoặc phản cảm",
		"dan_da":       "giọng miền quê chân chất, mộc mạc, từ ngữ bình dị và gần gũi; không giả giọng quá đà, không tự ý thêm thông tin địa phương hay công dụng sản phẩm",
	}[style]
	if description == "" {
		return "chuyên nghiệp, rõ ràng và đáng tin cậy"
	}
	return description
}

func hasCompleteSentenceEnding(script string) bool {
	trimmed := strings.TrimRight(strings.TrimSpace(script), "\"'”’»)]}")
	if trimmed == "" {
		return false
	}
	last := []rune(trimmed)[len([]rune(trimmed))-1]
	return strings.ContainsRune(".!?…。！？", last)
}
