package main

import "testing"

func TestParseCommand(t *testing.T) {
	tests := []struct {
		content string
		want    []string
	}{
		{
			content: "go run .",
			want:    []string{"go", "run", "."},
		},
		{
			content: "go build .",
			want:    []string{"go", "build", "."},
		},
		{
			content: "echo \"hello world\"",
			want:    []string{"echo", "hello world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			got, err := ParseCommand(tt.content)
			if err != nil {
				t.Errorf("ParseCommand() error = %v", err)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("ParseCommand() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseCommand() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
