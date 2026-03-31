package runner

import (
	"testing"
)

func TestFormatCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "echo",
			args:     []string{"hello", "world"},
			expected: "echo hello world",
		},
		{
			name:     "ls",
			args:     []string{"-l", "my folder"},
			expected: "ls -l \"my folder\"",
		},
		{
			name:     "empty",
			args:     []string{""},
			expected: "empty \"\"",
		},
		{
			name:     "with quotes",
			args:     []string{"hello \"world\""},
			expected: "\"with quotes\" \"hello \\\"world\\\"\"",
		},
	}

	for _, tt := range tests {
		actual := formatCommand(tt.name, tt.args)
		if actual != tt.expected {
			t.Errorf("formatCommand(%q, %v) = %q; expected %q", tt.name, tt.args, actual, tt.expected)
		}
	}
}
