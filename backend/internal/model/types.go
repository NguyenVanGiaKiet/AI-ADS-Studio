package model

import "time"

// UploadedVideo represents a source video uploaded by the user.
type UploadedVideo struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	Path      string    `json:"path"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
}

// RemixRequest holds parameters sent from frontend to generate remixed videos.
type RemixRequest struct {
	VideoIDs           []string `json:"videoIds"`
	OutputCount        int      `json:"outputCount"`
	Duration           int      `json:"duration"`
	AspectRatio        string   `json:"aspectRatio"`
	RemixMode          string   `json:"remixMode"`
	Deduplication      string   `json:"deduplication"`
	ReplaceVoice       bool     `json:"replaceVoice"`
	FollowSubtitles    bool     `json:"followSubtitles"`
	ProductDescription string   `json:"productDescription"`
	ScriptStyle        string   `json:"scriptStyle"`
	Voice              string   `json:"voice"`
	SpeechRate         float64  `json:"speechRate"`
}

// OutputVideo represents a remixed video produced by a remix job.
type OutputVideo struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"taskId"`
	Title     string    `json:"title"`
	Filename  string    `json:"filename"`
	URL       string    `json:"url"`
	Duration  int       `json:"duration"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
}

// RemixTask represents an async remix job state.
type RemixTask struct {
	ID                 string         `json:"id"`
	Status             string         `json:"status"` // pending, processing, completed, failed
	Progress           int            `json:"progress"` // 0 - 100
	Message            string         `json:"message"`
	Request            RemixRequest   `json:"request"`
	OutputVideos       []OutputVideo  `json:"outputVideos"`
	Error              string         `json:"error,omitempty"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}

// TTSPreviewRequest represents a request to preview AI TTS audio.
type TTSPreviewRequest struct {
	Text  string  `json:"text"`
	Voice string  `json:"voice"`
	Rate  float64 `json:"rate"`
}

// TTSPreviewResponse represents the result of a TTS preview request.
type TTSPreviewResponse struct {
	AudioURL string `json:"audioUrl"`
	Text     string `json:"text"`
	Voice    string `json:"voice"`
	Duration float64 `json:"duration"`
}
