package service

import "testing"

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