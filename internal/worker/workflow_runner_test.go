package worker

import (
	"testing"

	"github.com/surefire-ai/korus/internal/contract"
)

func TestIsWorkflowPattern(t *testing.T) {
	tests := []struct {
		name     string
		artifact contract.CompiledArtifact
		want     bool
	}{
		{
			name:     "empty artifact",
			artifact: contract.CompiledArtifact{},
			want:     false,
		},
		{
			name: "pattern.type workflow",
			artifact: contract.CompiledArtifact{
				Pattern: contract.ArtifactPattern{Type: "workflow"},
			},
			want: true,
		},
		{
			name: "runner.pattern.type workflow",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": "workflow"},
				},
			},
			want: true,
		},
		{
			name: "pattern.type react",
			artifact: contract.CompiledArtifact{
				Pattern: contract.ArtifactPattern{Type: "react"},
			},
			want: false,
		},
		{
			name: "both pattern.type and runner.pattern.type set to workflow",
			artifact: contract.CompiledArtifact{
				Pattern: contract.ArtifactPattern{Type: "workflow"},
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": "workflow"},
				},
			},
			want: true,
		},
		{
			name: "runner.pattern.type is non-string",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": 123},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWorkflowPattern(tt.artifact); got != tt.want {
				t.Errorf("isWorkflowPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}
