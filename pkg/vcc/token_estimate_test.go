package vcc

import "testing"

func TestEstimateTokensFromChars(t *testing.T) {
	tests := []struct {
		chars int
		want  int
	}{
		{0, 0},
		{-5, 0},
		{4, 1},   // 4 chars / 4 chars-per-token = 1 token
		{8, 2},
		{100, 25},
	}
	for _, tt := range tests {
		if got := estimateTokensFromChars(tt.chars); got != tt.want {
			t.Errorf("estimateTokensFromChars(%d) = %d, want %d", tt.chars, got, tt.want)
		}
	}
}

func TestClampCalibration(t *testing.T) {
	tests := []struct {
		in, want int
	}{
		{1, 2},
		{2, 2},
		{4, 4},
		{6, 6},
		{7, 6},
		{100, 6},
	}
	for _, tt := range tests {
		if got := clampCalibration(tt.in); got != tt.want {
			t.Errorf("clampCalibration(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestEstimateMessageContentCharsString(t *testing.T) {
	got := estimateMessageContentChars("hello")
	if got != 5 {
		t.Errorf("estimateMessageContentChars(string) = %d, want %d", got, 5)
	}
}

func TestEstimateMessageContentCharsEmpty(t *testing.T) {
	if got := estimateMessageContentChars(""); got != 0 {
		t.Errorf("estimateMessageContentChars(empty) = %d, want %d", got, 0)
	}
}

func TestEstimateMessageContentCharsImage(t *testing.T) {
	parts := []interface{}{
		map[string]interface{}{"image": true},
	}
	got := estimateMessageContentChars(parts)
	if got != 4800 {
		t.Errorf("estimateMessageContentChars(image) = %d, want %d", got, 4800)
	}
}

func TestEstimateMessageContentCharsMixed(t *testing.T) {
	parts := []interface{}{
		map[string]interface{}{"text": "hi"},    // 2
		map[string]interface{}{"image": true},   // 4800
		map[string]interface{}{"text": "bye"},   // 3
	}
	got := estimateMessageContentChars(parts)
	if got != 4805 {
		t.Errorf("estimateMessageContentChars(mixed) = %d, want %d", got, 4805)
	}
}
