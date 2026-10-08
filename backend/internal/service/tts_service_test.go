package service

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHasCompleteSentenceEnding(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   bool
	}{
		{name: "period", script: "Sản phẩm phù hợp với bạn.", want: true},
		{name: "exclamation with quote", script: "Chọn ngay hôm nay!\"", want: true},
		{name: "question", script: "Bạn đã sẵn sàng chưa?", want: true},
		{name: "truncated clause", script: "Sản phẩm phù hợp với", want: false},
		{name: "empty", script: "  ", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hasCompleteSentenceEnding(test.script); got != test.want {
				t.Fatalf("hasCompleteSentenceEnding(%q) = %t, want %t", test.script, got, test.want)
			}
		})
	}
}
func TestGetVoicesReturnsBundledVietnameseVoice(t *testing.T) {
	svc := NewTTSService(t.TempDir())
	voices, err := svc.GetVoices()
	if err != nil {
		t.Fatalf("GetVoices() unexpected error: %v", err)
	}
	if len(voices) == 0 {
		t.Fatal("GetVoices() returned no voices")
	}
	if voices[0].ID != freeVietnameseVoiceID {
		t.Fatalf("GetVoices()[0].ID = %q, want %q", voices[0].ID, freeVietnameseVoiceID)
	}
	if !strings.Contains(voices[0].Name, "Piper") {
		t.Fatalf("GetVoices()[0].Name = %q, want Piper voice label", voices[0].Name)
	}
}

func TestGetVoicesUsesConfiguredPiperModelName(t *testing.T) {
	modelPath := filepath.Join(t.TempDir(), "custom-vietnamese.onnx")
	if err := os.WriteFile(modelPath, []byte("model"), 0600); err != nil {
		t.Fatalf("create custom model fixture: %v", err)
	}
	svc := NewTTSService(t.TempDir())
	svc.VoiceModelPath = modelPath

	voices, err := svc.GetVoices()
	if err != nil {
		t.Fatalf("GetVoices() error = %v", err)
	}
	if len(voices) == 0 {
		t.Fatal("GetVoices() returned no configured voice")
	}
	if !strings.Contains(voices[0].Name, "custom-vietnamese") {
		t.Fatalf("GetVoices()[0] = %+v, want the configured model name", voices[0])
	}
}

func TestGetVoicesIncludesBundledPiperModels(t *testing.T) {
	svc := NewTTSService(t.TempDir())
	voices, err := svc.GetVoices()
	if err != nil {
		t.Fatalf("GetVoices() error = %v", err)
	}
	voiceIDs := make(map[string]bool, len(voices))
	for _, voice := range voices {
		voiceIDs[voice.ID] = true
	}
	for _, id := range []string{freeVietnameseVoiceID, "vi-vn-25hours-single", "vi-vn-vivos-0", "vi-vn-vivos-64"} {
		if !voiceIDs[id] {
			t.Errorf("GetVoices() did not include bundled voice %q", id)
		}
	}
	if len(voices) < 67 {
		t.Errorf("GetVoices() returned %d voices, want at least 67 bundled Piper voices", len(voices))
	}
}

func TestResolveVoiceUsesSelectedModelAndVivosSpeaker(t *testing.T) {
	svc := NewTTSService(t.TempDir())
	svc.VoiceModelPath = filepath.Join("custom", "default-voice.onnx")

	modelPath, speakerID, err := svc.resolveVoice(freeVietnameseVoiceID)
	if err != nil {
		t.Fatalf("resolveVoice(default) error = %v", err)
	}
	if modelPath != svc.VoiceModelPath || speakerID != nil {
		t.Fatalf("resolveVoice(default) = (%q, %v), want custom model and no speaker ID", modelPath, speakerID)
	}

	modelPath, speakerID, err = svc.resolveVoice("vi-vn-vivos-17")
	if err != nil {
		t.Fatalf("resolveVoice(VIVOS) error = %v", err)
	}
	if filepath.Base(modelPath) != vietnameseVivosModel || speakerID == nil || *speakerID != 17 {
		t.Fatalf("resolveVoice(VIVOS) = (%q, %v), want VIVOS model and speaker 17", modelPath, speakerID)
	}
}

func TestGenerateScriptRequiresGroqAPIKey(t *testing.T) {
	svc := NewTTSService(t.TempDir())

	script, err := svc.GenerateScript("Áo khoác màu xanh", "friendly", 15, 1)
	if err == nil {
		t.Fatal("GenerateScript() expected an error when the Groq API key is missing")
	}
	if !strings.Contains(err.Error(), "GROQ_API_KEY") {
		t.Fatalf("GenerateScript() error = %q, want GROQ_API_KEY configuration guidance", err)
	}
	if script != "" {
		t.Fatalf("GenerateScript() script = %q, want empty script on error", script)
	}
}

func TestVoicePreviewUsesFixedStudioIntroduction(t *testing.T) {
	const want = "AI ADS Studio là nền tảng AI giúp tự động hóa quy trình tạo video quảng cáo chuyên nghiệp từ hình ảnh và thông tin sản phẩm, nhanh chóng, dễ dàng và tiết kiệm chi phí."
	if voicePreviewText != want {
		t.Fatalf("voicePreviewText = %q, want %q", voicePreviewText, want)
	}
}

func TestScriptStyleDescriptions(t *testing.T) {
	tests := []struct {
		style string
		want  string
	}{
		{style: "professional", want: "chuyên nghiệp"},
		{style: "adam_drama", want: "drama đời thường"},
		{style: "adam_viral", want: "cảm thán mạnh"},
		{style: "dan_da", want: "chân chất, mộc mạc"},
	}
	for _, test := range tests {
		if got := scriptStyleDescription(test.style); !strings.Contains(got, test.want) {
			t.Errorf("scriptStyleDescription(%q) = %q, want it to contain %q", test.style, got, test.want)
		}
	}
	if got := scriptStyleDescription("unknown"); got != scriptStyleDescription("professional") {
		t.Errorf("unknown style description = %q, want professional fallback %q", got, scriptStyleDescription("professional"))
	}
}

func TestGenerateSpeechRejectsUnknownVoice(t *testing.T) {
	svc := NewTTSService(t.TempDir())
	if _, err := svc.GenerateSpeech("Xin chào.", "remote-voice", 1); err == nil {
		t.Fatal("GenerateSpeech() expected an error for a voice that is not bundled")
	}
}

func TestFitTempoFactorMatchesTargetDuration(t *testing.T) {
	sourceDuration := 12.0
	targetDuration := 10.0
	tempo := fitTempoFactor(sourceDuration, targetDuration, 1)
	fittedDuration := sourceDuration / tempo
	if fittedDuration > targetDuration || targetDuration-fittedDuration > 0.1 {
		t.Fatalf("fitted duration = %.3f seconds, want within 0.1 seconds and not longer than target %.3f", fittedDuration, targetDuration)
	}
}

func TestFitTempoFactorFitsSpeechToTargetDuration(t *testing.T) {
	tests := []struct {
		name          string
		source        float64
		target        float64
		requestedRate float64
		want          float64
	}{
		{name: "slow speech to fill target duration", source: 8, target: 10, requestedRate: 1, want: 0.804},
		{name: "target fit takes precedence over requested rate", source: 8, target: 10, requestedRate: 1.3, want: 0.804},
		{name: "speed up speech that exceeds target", source: 12, target: 10, requestedRate: 0.7, want: 1.206},
		{name: "respect selected rate without duration target", source: 0, target: 0, requestedRate: 1.3, want: 1.3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := fitTempoFactor(test.source, test.target, test.requestedRate); math.Abs(got-test.want) > 0.001 {
				t.Fatalf("fitTempoFactor() = %.3f, want %.3f", got, test.want)
			}
		})
	}
}

func TestScriptTargetWordsUsesCalibratedVoiceRateAndSelectedSpeed(t *testing.T) {
	tests := []struct {
		name      string
		duration  float64
		speed     float64
		voiceRate float64
		want      int
	}{
		{name: "normal voice speed", duration: 30, speed: 1, voiceRate: 4, want: 120},
		{name: "selected slower speech", duration: 30, speed: 0.7, voiceRate: 4, want: 84},
		{name: "selected faster speech", duration: 30, speed: 1.3, voiceRate: 4, want: 156},
		{name: "cap long scripts", duration: 60, speed: 1.3, voiceRate: 8, want: 500},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := scriptTargetWords(test.duration, test.speed, test.voiceRate); got != test.want {
				t.Fatalf("scriptTargetWords() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestBuildAudioTempoFilterSplitsOutOfRangeFactors(t *testing.T) {
	tests := []struct {
		tempo float64
		want  string
	}{
		{tempo: 4, want: "atempo=2.000,atempo=2.000000"},
		{tempo: 0.25, want: "atempo=0.500,atempo=0.500000"},
	}
	for _, test := range tests {
		if got := buildAudioTempoFilter(test.tempo); got != test.want {
			t.Errorf("buildAudioTempoFilter(%v) = %q, want %q", test.tempo, got, test.want)
		}
	}
}

func TestEstimateSyllablesPerSecondUsesSelectedPiperVoice(t *testing.T) {
	modelPath := findBundledVoiceModel()
	if _, err := os.Stat(modelPath); err != nil {
		t.Skip("bundled Vietnamese Piper model is unavailable")
	}
	for _, executable := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(executable); err != nil {
			t.Skipf("%s is unavailable", executable)
		}
	}
	python := os.Getenv("PIPER_PYTHON")
	if python == "" {
		python = "python"
	}
	if err := exec.Command(python, "-m", "piper", "--help").Run(); err != nil {
		t.Skip("Piper runtime is unavailable")
	}

	svc := NewTTSService(t.TempDir())
	svc.PiperPython = python
	svc.VoiceModelPath = modelPath
	rate, err := svc.estimateSyllablesPerSecond(freeVietnameseVoiceID)
	if err != nil {
		t.Fatalf("estimateSyllablesPerSecond() error = %v", err)
	}
	if rate <= 0 {
		t.Fatalf("estimateSyllablesPerSecond() = %f, want positive measured rate", rate)
	}
	if cachedRate, err := svc.estimateSyllablesPerSecond(freeVietnameseVoiceID); err != nil || cachedRate != rate {
		t.Fatalf("cached estimate = %f, %v; want %f, nil", cachedRate, err, rate)
	}
}

func TestGenerateSpeechRateMaximumIsOnePointThree(t *testing.T) {
	svc := NewTTSService(t.TempDir())
	svc.VoiceModelPath = filepath.Join(t.TempDir(), "missing-model.onnx")
	_, err := svc.GenerateSpeechForDuration("Xin chào.", freeVietnameseVoiceID, 1.3, 0)
	if err == nil || !strings.Contains(err.Error(), "missing-model.onnx") {
		t.Fatalf("GenerateSpeechForDuration() error = %v, want model lookup error (rate 1.3 should pass validation)", err)
	}
}

func TestGenerateSpeechForDurationFitsGeneratedAudio(t *testing.T) {
	modelPath := findBundledVoiceModel()
	if _, err := os.Stat(modelPath); err != nil {
		t.Skip("bundled Vietnamese Piper model is unavailable")
	}
	for _, executable := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(executable); err != nil {
			t.Skipf("%s is unavailable", executable)
		}
	}
	python := os.Getenv("PIPER_PYTHON")
	if python == "" {
		python = "python"
	}
	if err := exec.Command(python, "-m", "piper", "--help").Run(); err != nil {
		t.Skip("Piper runtime is unavailable")
	}

	svc := NewTTSService(t.TempDir())
	svc.PiperPython = python
	svc.VoiceModelPath = modelPath
	const targetDuration = 4.0
	for _, voiceID := range []string{freeVietnameseVoiceID, "vi-vn-25hours-single", "vi-vn-vivos-17"} {
		t.Run(voiceID, func(t *testing.T) {
			audioPath, err := svc.GenerateSpeechForDuration("Xin chào, đây là lời giới thiệu sản phẩm.", voiceID, 1, targetDuration)
			if err != nil {
				t.Fatalf("GenerateSpeechForDuration() error = %v", err)
			}
			actualDuration, err := probeDuration(audioPath)
			if err != nil {
				t.Fatalf("probe generated audio duration: %v", err)
			}
			if math.Abs(actualDuration-targetDuration) > 0.05 {
				t.Fatalf("generated audio duration = %.3f, want %.3f ± 0.05 seconds", actualDuration, targetDuration)
			}
		})
	}
}
