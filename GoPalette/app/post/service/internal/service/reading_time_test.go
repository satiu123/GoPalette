package service

import (
	"strings"
	"testing"
)

func TestEstimateReadingMinutes(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		fallback string
		want     int64
	}{
		{
			name:    "minimum one minute",
			content: "短文",
			want:    1,
		},
		{
			name:    "cjk rounds up at 500 characters per minute",
			content: strings.Repeat("中", 501),
			want:    2,
		},
		{
			name:    "english words use 225 words per minute",
			content: strings.Repeat("word ", 226),
			want:    2,
		},
		{
			name:     "fallback summary is used when content is empty",
			fallback: strings.Repeat("字", 600),
			want:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := estimateReadingMinutes(tt.content, tt.fallback); got != tt.want {
				t.Fatalf("estimateReadingMinutes() = %d, want %d", got, tt.want)
			}
		})
	}
}
