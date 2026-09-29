package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"ai-ads-studio/backend/internal/model"
)

type TTSService struct {
	AudioDir string
}

func NewTTSService(audioDir string) *TTSService {
	os.MkdirAll(audioDir, 0755)
	return &TTSService{AudioDir: audioDir}
}

// GeneratePreview creates a preview audio file or returns preview metadata.
func (s *TTSService) GeneratePreview(req model.TTSPreviewRequest) (*model.TTSPreviewResponse, error) {
	if req.Text == "" {
		req.Text = "Xin chào, đây là phần nghe thử giọng đọc quảng cáo của bạn."
	}
	if req.Voice == "" {
		req.Voice = "Ngọc Huyền (Vbee)"
	}
	if req.Rate <= 0 {
		req.Rate = 1.0
	}

	hash := md5.Sum([]byte(fmt.Sprintf("%s_%s_%.1f", req.Text, req.Voice, req.Rate)))
	filename := fmt.Sprintf("preview_%s.wav", hex.EncodeToString(hash[:8]))
	audioPath := filepath.Join(s.AudioDir, filename)

	// Create dummy audio metadata or file if not exists
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		dummyAudioHeader := []byte("RIFF4\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x01\x00D\xac\x00\x00\x88X\x01\x00\x02\x00\x10\x00data\x00\x00\x00\x00")
		_ = os.WriteFile(audioPath, dummyAudioHeader, 0644)
	}

	audioURL := "/storage/tts/" + filename
	return &model.TTSPreviewResponse{
		AudioURL: audioURL,
		Text:     req.Text,
		Voice:    req.Voice,
		Duration: 3.5 / req.Rate,
	}, nil
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
