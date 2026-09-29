package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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
	return &TTSService{AudioDir: audioDir, ModelDir: modelDir, Executable: executable}
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
		req.Text = "Xin chào, đây là phần nghe thử giọng đọc quảng cáo của bạn."
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

	audioPath, err := s.GenerateSpeech(req.Text, req.Voice, req.Rate)
	if err != nil {
		return nil, err
	}
	filename := filepath.Base(audioPath)
	audioURL := "/storage/tts/" + filename
	return &model.TTSPreviewResponse{
		AudioURL: audioURL,
		Text:     req.Text,
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

// GenerateScript generates an advertising script from product description and style.
func (s *TTSService) GenerateScript(productDescription string, style string) string {
	if productDescription == "" {
		return "Sản phẩm chất lượng cao, ưu đãi cực sốc hôm nay. Mua ngay kẻo lỡ!"
	}

	switch style {
	case "friendly":
		return fmt.Sprintf("Chào cả nhà nha! Hôm nay mình giới thiệu cho mọi người %s. Thật sự dùng quá mê luôn, đặt mua ngay hôm nay để nhận ưu đãi nhé!", productDescription)
	case "energetic":
		return fmt.Sprintf("SIÊU HOT! %s ĐÃ CÓ MẶT! Giá cực hời, mua 1 được 2, số lượng có hạn! Bấm vào giỏ hàng ngay!", productDescription)
	case "storytelling":
		return fmt.Sprintf("Có những trải nghiệm làm bạn bất ngờ từ lần đầu tiên. %s chính là câu chuyện đó. Hãy trải nghiệm sự khác biệt ngay hôm nay.", productDescription)
	default: // professional
		return fmt.Sprintf("Giải pháp hoàn hảo cho nhu cầu của bạn: %s. Thiết kế tối ưu, cam kết chất lượng hàng đầu. Đặt hàng ngay hôm nay.", productDescription)
	}
}
