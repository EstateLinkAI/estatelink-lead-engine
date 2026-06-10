package strategy

import "testing"

func TestGradeFromScore(t *testing.T) {
	tests := []struct {
		name  string
		score int
		want  string
	}{
		{name: "A grade", score: 90, want: "A"},
		{name: "B grade", score: 75, want: "B"},
		{name: "C grade", score: 60, want: "C"},
		{name: "D grade", score: 45, want: "D"},
		{name: "E grade", score: 20, want: "E"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GradeFromScore(tt.score)
			if got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}